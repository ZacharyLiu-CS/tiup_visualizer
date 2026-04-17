package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

// --- Region/Peer types for JSON parsing from tiup ctl output ---

type PDPeer struct {
	ID       uint64 `json:"id"`
	StoreID  uint64 `json:"store_id"`
	RoleName string `json:"role_name"`
}

type PDRegion struct {
	ID              uint64   `json:"id"`
	StartKey        string   `json:"start_key"`
	EndKey          string   `json:"end_key"`
	Peers           []PDPeer `json:"peers"`
	Leader          *PDPeer  `json:"leader"`
	ApproximateSize int64    `json:"approximate_size"`
}

type PDRegionResponse struct {
	Count   int        `json:"count"`
	Regions []PDRegion `json:"regions"`
}

// --- Balance operation types ---

type BalanceOpType string

const (
	OpTransferPeer   BalanceOpType = "transfer-peer"
	OpTransferLeader BalanceOpType = "transfer-leader"
)

type BalanceOperation struct {
	Type      BalanceOpType `json:"type"`
	RegionID  uint64        `json:"region_id"`
	FromStore uint64        `json:"from_store,omitempty"`
	ToStore   uint64        `json:"to_store"`
	Reason    string        `json:"reason"`
}

type StoreDistribution struct {
	StoreID     uint64  `json:"store_id"`
	PeerCount   int     `json:"peer_count"`
	LeaderCount int     `json:"leader_count"`
	PeerDelta   float64 `json:"peer_delta"`
	LeaderDelta float64 `json:"leader_delta"`
}

type AnalyzeResult struct {
	TotalRegions int                 `json:"total_regions"`
	TotalStores  int                 `json:"total_stores"`
	TotalPeers   int                 `json:"total_peers"`
	TotalLeaders int                 `json:"total_leaders"`
	IdealPeers   float64             `json:"ideal_peers"`
	IdealLeaders float64             `json:"ideal_leaders"`
	Before       []StoreDistribution `json:"before"`
	After        []StoreDistribution `json:"after"`
	Operations   []BalanceOperation  `json:"operations"`
	PeerOps      int                 `json:"peer_ops"`
	LeaderOps    int                 `json:"leader_ops"`
}

// --- Task types ---

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskCancelled TaskStatus = "cancelled"
	TaskFailed    TaskStatus = "failed"
)

type TaskConfig struct {
	PDAddr          string `json:"pd_addr"`
	TiUPVersion     string `json:"tiup_version"`
	PeerThreshold   int    `json:"peer_threshold"`
	LeaderThreshold int    `json:"leader_threshold"`
	BatchSize       int    `json:"batch_size"`
}

type OpResult struct {
	Operation BalanceOperation `json:"operation"`
	Status    string           `json:"status"` // "success", "failed", "skipped"
	Error     string           `json:"error,omitempty"`
	Duration  string           `json:"duration,omitempty"`
}

type BalanceTask struct {
	ID         string         `json:"id"`
	Status     TaskStatus     `json:"status"`
	Config     TaskConfig     `json:"config"`
	Plan       *AnalyzeResult `json:"plan,omitempty"`
	Progress   int            `json:"progress"`
	Total      int            `json:"total"`
	Results    []OpResult     `json:"results"`
	Error      string         `json:"error,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	StartedAt  *time.Time     `json:"started_at,omitempty"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`

	cancel context.CancelFunc `json:"-"`
}

// --- SSE event ---

type SSEEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// --- BalancerService ---

type BalancerService struct {
	mu          sync.RWMutex
	tasks       map[string]*BalanceTask
	taskOrder   []string
	concurrency int
	taskCh      chan string
	sseClients  map[chan SSEEvent]bool
	sseMu       sync.RWMutex
	workerCtx   context.Context
	stopWorkers context.CancelFunc
}

func NewBalancerService() *BalancerService {
	ctx, cancel := context.WithCancel(context.Background())
	s := &BalancerService{
		tasks:       make(map[string]*BalanceTask),
		concurrency: 1,
		taskCh:      make(chan string, 256),
		sseClients:  make(map[chan SSEEvent]bool),
		workerCtx:   ctx,
		stopWorkers: cancel,
	}
	s.startWorkers(ctx, s.concurrency)
	return s
}

func (s *BalancerService) Stop() {
	if s.stopWorkers != nil {
		s.stopWorkers()
	}
}

// --- SSE subscription ---

func (s *BalancerService) Subscribe() chan SSEEvent {
	ch := make(chan SSEEvent, 32)
	s.sseMu.Lock()
	s.sseClients[ch] = true
	s.sseMu.Unlock()
	return ch
}

func (s *BalancerService) Unsubscribe(ch chan SSEEvent) {
	s.sseMu.Lock()
	delete(s.sseClients, ch)
	s.sseMu.Unlock()
	close(ch)
}

func (s *BalancerService) broadcast(event SSEEvent) {
	s.sseMu.RLock()
	defer s.sseMu.RUnlock()
	for ch := range s.sseClients {
		select {
		case ch <- event:
		default:
			// drop if client is slow
		}
	}
}

// --- Worker Pool ---

func (s *BalancerService) startWorkers(ctx context.Context, n int) {
	for i := 0; i < n; i++ {
		go func(workerID int) {
			slog.Info("Balancer worker started", "worker", workerID)
			for {
				select {
				case <-ctx.Done():
					slog.Info("Balancer worker stopped", "worker", workerID)
					return
				case taskID, ok := <-s.taskCh:
					if !ok {
						return
					}
					s.executeTask(ctx, taskID)
				}
			}
		}(i)
	}
}

func (s *BalancerService) SetConcurrency(n int) {
	if n < 1 {
		n = 1
	}
	if n > 20 {
		n = 20
	}
	s.mu.Lock()
	oldConcurrency := s.concurrency
	s.concurrency = n
	s.mu.Unlock()

	if n == oldConcurrency {
		return
	}

	// Stop old workers and start new ones
	s.stopWorkers()
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.workerCtx = ctx
	s.stopWorkers = cancel
	s.mu.Unlock()
	s.startWorkers(ctx, n)
	slog.Info("Balancer concurrency changed", "old", oldConcurrency, "new", n)
}

func (s *BalancerService) GetConcurrency() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.concurrency
}

// --- Task CRUD ---

func (s *BalancerService) CreateTask(config TaskConfig) (string, error) {
	if config.PDAddr == "" {
		return "", fmt.Errorf("pd_addr is required")
	}
	if config.TiUPVersion == "" {
		config.TiUPVersion = "v8.1.0"
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 5
	}
	if config.PeerThreshold <= 0 {
		config.PeerThreshold = 3
	}
	if config.LeaderThreshold <= 0 {
		config.LeaderThreshold = 2
	}

	// Analyze first to get the plan
	plan, err := s.Analyze(config.PDAddr, config.TiUPVersion, config.PeerThreshold, config.LeaderThreshold)
	if err != nil {
		return "", fmt.Errorf("failed to analyze: %w", err)
	}

	id := fmt.Sprintf("bal-%d-%04d", time.Now().Unix(), rand.Intn(10000))
	now := time.Now()

	task := &BalanceTask{
		ID:        id,
		Status:    TaskPending,
		Config:    config,
		Plan:      plan,
		Progress:  0,
		Total:     len(plan.Operations),
		Results:   []OpResult{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	s.tasks[id] = task
	s.taskOrder = append(s.taskOrder, id)
	s.mu.Unlock()

	s.broadcast(SSEEvent{Type: "task_created", Data: task})

	// Enqueue for workers
	select {
	case s.taskCh <- id:
	default:
		slog.Warn("Task channel full, task may be delayed", "task_id", id)
	}

	slog.Info("Balance task created", "task_id", id, "operations", task.Total)
	return id, nil
}

func (s *BalancerService) GetTask(taskID string) *BalanceTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[taskID]
	if !ok {
		return nil
	}
	return task
}

func (s *BalancerService) ListTasks() []*BalanceTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*BalanceTask, 0, len(s.taskOrder))
	for _, id := range s.taskOrder {
		if t, ok := s.tasks[id]; ok {
			result = append(result, t)
		}
	}
	return result
}

func (s *BalancerService) CancelTask(taskID string) error {
	s.mu.Lock()
	task, ok := s.tasks[taskID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("task %q not found", taskID)
	}
	if task.Status != TaskPending && task.Status != TaskRunning {
		s.mu.Unlock()
		return fmt.Errorf("task %q is %s, cannot cancel", taskID, task.Status)
	}
	if task.cancel != nil {
		task.cancel()
	}
	task.Status = TaskCancelled
	now := time.Now()
	task.UpdatedAt = now
	task.FinishedAt = &now
	s.mu.Unlock()

	s.broadcast(SSEEvent{Type: "task_update", Data: task})
	slog.Info("Balance task cancelled", "task_id", taskID)
	return nil
}

func (s *BalancerService) DeleteTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %q not found", taskID)
	}
	if task.Status == TaskPending || task.Status == TaskRunning {
		return fmt.Errorf("task %q is %s, cancel it first", taskID, task.Status)
	}
	delete(s.tasks, taskID)
	// Remove from order
	for i, id := range s.taskOrder {
		if id == taskID {
			s.taskOrder = append(s.taskOrder[:i], s.taskOrder[i+1:]...)
			break
		}
	}
	s.broadcast(SSEEvent{Type: "task_deleted", Data: map[string]string{"id": taskID}})
	slog.Info("Balance task deleted", "task_id", taskID)
	return nil
}

// --- Task Execution ---

func (s *BalancerService) executeTask(workerCtx context.Context, taskID string) {
	s.mu.Lock()
	task, ok := s.tasks[taskID]
	if !ok {
		s.mu.Unlock()
		return
	}
	if task.Status == TaskCancelled {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(workerCtx)
	task.cancel = cancel
	task.Status = TaskRunning
	now := time.Now()
	task.StartedAt = &now
	task.UpdatedAt = now
	s.mu.Unlock()

	s.broadcast(SSEEvent{Type: "task_update", Data: task})

	defer cancel()

	config := task.Config
	operations := task.Plan.Operations
	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 5
	}

	totalOps := len(operations)
	failed := false

	for i := 0; i < totalOps; i += batchSize {
		// Check cancellation
		select {
		case <-ctx.Done():
			s.markTaskCancelled(task)
			return
		default:
		}

		end := i + batchSize
		if end > totalOps {
			end = totalOps
		}
		batch := operations[i:end]

		batchNum := i/batchSize + 1
		totalBatches := (totalOps + batchSize - 1) / batchSize
		slog.Info(fmt.Sprintf("Task %s: executing batch %d/%d (%d ops)", taskID, batchNum, totalBatches, len(batch)))

		for _, op := range batch {
			select {
			case <-ctx.Done():
				s.markTaskCancelled(task)
				return
			default:
			}

			opResult := s.executeOperation(ctx, config, op)

			s.mu.Lock()
			task.Results = append(task.Results, opResult)
			task.Progress = len(task.Results)
			task.UpdatedAt = time.Now()
			s.mu.Unlock()

			if opResult.Status == "failed" {
				slog.Warn("Operation failed", "task_id", taskID, "op", opResult.Operation.Type, "region", opResult.Operation.RegionID, "error", opResult.Error)
			}

			s.broadcast(SSEEvent{Type: "task_update", Data: task})
		}

		// Wait for batch completion by polling operator check
		s.waitBatchCompletion(ctx, config, batch)
	}

	// Finalize
	s.mu.Lock()
	finishTime := time.Now()
	task.FinishedAt = &finishTime
	task.UpdatedAt = finishTime
	if failed {
		task.Status = TaskFailed
	} else {
		task.Status = TaskCompleted
	}
	// Check if any ops failed
	for _, r := range task.Results {
		if r.Status == "failed" {
			task.Status = TaskCompleted // still completed, with some failures noted
			break
		}
	}
	s.mu.Unlock()

	s.broadcast(SSEEvent{Type: "task_update", Data: task})
	slog.Info("Balance task finished", "task_id", taskID, "status", task.Status, "progress", task.Progress, "total", task.Total)
}

func (s *BalancerService) markTaskCancelled(task *BalanceTask) {
	s.mu.Lock()
	if task.Status == TaskRunning {
		task.Status = TaskCancelled
		now := time.Now()
		task.FinishedAt = &now
		task.UpdatedAt = now
	}
	s.mu.Unlock()
	s.broadcast(SSEEvent{Type: "task_update", Data: task})
}

func (s *BalancerService) executeOperation(ctx context.Context, config TaskConfig, op BalanceOperation) OpResult {
	start := time.Now()

	var cmdStr string
	switch op.Type {
	case OpTransferPeer:
		cmdStr = fmt.Sprintf("tiup ctl:%s pd -u %s operator add transfer-peer %d %d %d",
			config.TiUPVersion, config.PDAddr, op.RegionID, op.FromStore, op.ToStore)
	case OpTransferLeader:
		cmdStr = fmt.Sprintf("tiup ctl:%s pd -u %s operator add transfer-leader %d %d",
			config.TiUPVersion, config.PDAddr, op.RegionID, op.ToStore)
	default:
		return OpResult{Operation: op, Status: "failed", Error: "unknown operation type"}
	}

	slog.Info("Balancer: executing operation", "type", op.Type, "region", op.RegionID, "command", cmdStr)
	output, err := runTiUPCommand(ctx, cmdStr, 30*time.Second)
	duration := time.Since(start).Round(time.Millisecond).String()

	if err != nil {
		slog.Error("Balancer: operation failed", "type", op.Type, "region", op.RegionID, "error", err, "output", strings.TrimSpace(output), "duration", duration)
		return OpResult{
			Operation: op,
			Status:    "failed",
			Error:     fmt.Sprintf("%v (output: %s)", err, strings.TrimSpace(output)),
			Duration:  duration,
		}
	}

	slog.Info("Balancer: operation succeeded", "type", op.Type, "region", op.RegionID, "duration", duration)
	return OpResult{
		Operation: op,
		Status:    "success",
		Duration:  duration,
	}
}

func (s *BalancerService) waitBatchCompletion(ctx context.Context, config TaskConfig, batch []BalanceOperation) {
	deadline := time.After(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	pending := make(map[uint64]bool)
	for _, op := range batch {
		pending[op.RegionID] = true
	}

	for len(pending) > 0 {
		select {
		case <-ctx.Done():
			return
		case <-deadline:
			slog.Warn("Batch completion timeout, proceeding", "remaining", len(pending))
			return
		case <-ticker.C:
			for regionID := range pending {
				cmdStr := fmt.Sprintf("tiup ctl:%s pd -u %s operator check %d",
					config.TiUPVersion, config.PDAddr, regionID)
				output, err := runTiUPCommand(ctx, cmdStr, 15*time.Second)
				if err != nil {
					continue
				}
				output = strings.TrimSpace(output)
				if output == "" {
					delete(pending, regionID)
				}
			}
		}
	}
}

// --- Analyze (dry-run) ---

func (s *BalancerService) Analyze(pdAddr, tiupVersion string, peerThreshold, leaderThreshold int) (*AnalyzeResult, error) {
	if pdAddr == "" {
		return nil, fmt.Errorf("pd_addr is required")
	}
	if tiupVersion == "" {
		tiupVersion = "v8.1.0"
	}
	if peerThreshold < 0 {
		peerThreshold = 3
	}
	if leaderThreshold < 0 {
		leaderThreshold = 2
	}

	// Fetch region data
	cmdStr := fmt.Sprintf("tiup ctl:%s pd -u %s region", tiupVersion, pdAddr)
	slog.Info("Balancer: fetching region data", "pd_addr", pdAddr, "tiup_version", tiupVersion, "command", cmdStr)
	output, err := runTiUPCommand(context.Background(), cmdStr, 60*time.Second)
	if err != nil {
		slog.Error("Balancer: failed to fetch regions", "error", err, "output", balancerTruncate(output, 500))
		return nil, fmt.Errorf("failed to fetch regions: %w (output: %s)", err, balancerTruncate(output, 200))
	}

	// tiup ctl may print banner/progress text before the JSON output.
	// Extract the JSON portion by finding the first '{' character.
	jsonOutput := extractJSON(output)
	if jsonOutput == "" {
		slog.Error("Balancer: no JSON found in tiup output", "output", balancerTruncate(output, 500))
		return nil, fmt.Errorf("no JSON found in tiup ctl output, raw output: %s", balancerTruncate(output, 300))
	}

	var resp PDRegionResponse
	if err := json.Unmarshal([]byte(jsonOutput), &resp); err != nil {
		slog.Error("Balancer: failed to parse region JSON", "error", err, "output_prefix", balancerTruncate(jsonOutput, 200))
		return nil, fmt.Errorf("failed to parse region JSON: %w", err)
	}

	if len(resp.Regions) == 0 {
		return nil, fmt.Errorf("no regions found")
	}

	// Compute before-stats
	beforeStats := computeDistribution(resp.Regions)
	numStores := len(beforeStats)
	if numStores == 0 {
		return nil, fmt.Errorf("no stores found")
	}

	totalPeers := 0
	totalLeaders := 0
	for _, st := range beforeStats {
		totalPeers += st.PeerCount
		totalLeaders += st.LeaderCount
	}
	idealPeers := float64(totalPeers) / float64(numStores)
	idealLeaders := float64(totalLeaders) / float64(numStores)

	// Fill deltas
	beforeDist := makeDistSlice(beforeStats, idealPeers, idealLeaders)

	// Deep copy regions for mutation
	workRegions := deepCopyPDRegions(resp.Regions)

	// Phase 1: balance peers
	peerOps := balancePeers(workRegions, beforeStats, idealPeers, float64(peerThreshold))

	// Phase 2: balance leaders
	leaderOps := balanceLeaders(workRegions, idealLeaders, float64(leaderThreshold))

	allOps := append(peerOps, leaderOps...)
	if allOps == nil {
		allOps = []BalanceOperation{}
	}

	// Compute after-stats
	afterStats := computeDistribution(workRegions)
	afterDist := makeDistSlice(afterStats, idealPeers, idealLeaders)

	return &AnalyzeResult{
		TotalRegions: len(resp.Regions),
		TotalStores:  numStores,
		TotalPeers:   totalPeers,
		TotalLeaders: totalLeaders,
		IdealPeers:   math.Round(idealPeers*100) / 100,
		IdealLeaders: math.Round(idealLeaders*100) / 100,
		Before:       beforeDist,
		After:        afterDist,
		Operations:   allOps,
		PeerOps:      len(peerOps),
		LeaderOps:    len(leaderOps),
	}, nil
}

// --- Balance Algorithm ---

type storeStats struct {
	storeID     uint64
	PeerCount   int
	LeaderCount int
}

func computeDistribution(regions []PDRegion) map[uint64]*storeStats {
	stats := make(map[uint64]*storeStats)
	for _, region := range regions {
		for _, peer := range region.Peers {
			if peer.RoleName != "Voter" && peer.RoleName != "" {
				continue
			}
			st, ok := stats[peer.StoreID]
			if !ok {
				st = &storeStats{storeID: peer.StoreID}
				stats[peer.StoreID] = st
			}
			st.PeerCount++
		}
		if region.Leader != nil {
			st, ok := stats[region.Leader.StoreID]
			if !ok {
				st = &storeStats{storeID: region.Leader.StoreID}
				stats[region.Leader.StoreID] = st
			}
			st.LeaderCount++
		}
	}
	return stats
}

func makeDistSlice(stats map[uint64]*storeStats, idealPeers, idealLeaders float64) []StoreDistribution {
	dist := make([]StoreDistribution, 0, len(stats))
	for _, st := range stats {
		dist = append(dist, StoreDistribution{
			StoreID:     st.storeID,
			PeerCount:   st.PeerCount,
			LeaderCount: st.LeaderCount,
			PeerDelta:   math.Round((float64(st.PeerCount)-idealPeers)*100) / 100,
			LeaderDelta: math.Round((float64(st.LeaderCount)-idealLeaders)*100) / 100,
		})
	}
	sort.Slice(dist, func(i, j int) bool { return dist[i].StoreID < dist[j].StoreID })
	return dist
}

func balancePeers(regions []PDRegion, stats map[uint64]*storeStats, ideal, threshold float64) []BalanceOperation {
	storePeerCount := make(map[uint64]int)
	for id, st := range stats {
		storePeerCount[id] = st.PeerCount
	}

	storeRegionIdx := buildPDStoreRegionIndex(regions)
	var ops []BalanceOperation

	totalPeers := 0
	for _, c := range storePeerCount {
		totalPeers += c
	}

	for iter := 0; iter < totalPeers; iter++ {
		overloaded, underloaded := classifyPDStores(storePeerCount, ideal, threshold)
		if len(overloaded) == 0 || len(underloaded) == 0 {
			break
		}

		moved := false
		for _, src := range overloaded {
			if moved {
				break
			}
			for _, dst := range underloaded {
				regionID, ok := findPDRegionToTransferPeer(regions, storeRegionIdx, src, dst)
				if !ok {
					continue
				}
				ops = append(ops, BalanceOperation{
					Type:      OpTransferPeer,
					RegionID:  regionID,
					FromStore: src,
					ToStore:   dst,
					Reason:    fmt.Sprintf("move peer from store %d (%d peers) to store %d (%d peers)", src, storePeerCount[src], dst, storePeerCount[dst]),
				})
				transferPDPeerInRegions(regions, regionID, src, dst)
				storeRegionIdx[src].remove(regionID)
				if storeRegionIdx[dst] == nil {
					storeRegionIdx[dst] = newPDRegionSet()
				}
				storeRegionIdx[dst].add(regionID)
				storePeerCount[src]--
				storePeerCount[dst]++
				moved = true
				break
			}
		}
		if !moved {
			break
		}
	}
	return ops
}

func balanceLeaders(regions []PDRegion, ideal, threshold float64) []BalanceOperation {
	stats := computeDistribution(regions)
	storeLeaderCount := make(map[uint64]int)
	for id, st := range stats {
		storeLeaderCount[id] = st.LeaderCount
	}

	var ops []BalanceOperation
	totalLeaders := 0
	for _, c := range storeLeaderCount {
		totalLeaders += c
	}

	for iter := 0; iter < totalLeaders; iter++ {
		overloaded, underloaded := classifyPDStores(storeLeaderCount, ideal, threshold)
		if len(overloaded) == 0 || len(underloaded) == 0 {
			break
		}

		moved := false
		for _, src := range overloaded {
			if moved {
				break
			}
			for _, dst := range underloaded {
				regionID, ok := findPDRegionToTransferLeader(regions, src, dst)
				if !ok {
					continue
				}
				ops = append(ops, BalanceOperation{
					Type:     OpTransferLeader,
					RegionID: regionID,
					ToStore:  dst,
					Reason:   fmt.Sprintf("move leader from store %d (%d leaders) to store %d (%d leaders)", src, storeLeaderCount[src], dst, storeLeaderCount[dst]),
				})
				transferPDLeaderInRegions(regions, regionID, dst)
				storeLeaderCount[src]--
				storeLeaderCount[dst]++
				moved = true
				break
			}
		}
		if !moved {
			break
		}
	}
	return ops
}

func classifyPDStores(storeCounts map[uint64]int, ideal, threshold float64) (overloaded, underloaded []uint64) {
	for storeID, count := range storeCounts {
		diff := float64(count) - ideal
		if diff > threshold {
			overloaded = append(overloaded, storeID)
		} else if diff < -threshold {
			underloaded = append(underloaded, storeID)
		}
	}
	sort.Slice(overloaded, func(i, j int) bool {
		return storeCounts[overloaded[i]] > storeCounts[overloaded[j]]
	})
	sort.Slice(underloaded, func(i, j int) bool {
		return storeCounts[underloaded[i]] < storeCounts[underloaded[j]]
	})
	return
}

func findPDRegionToTransferPeer(regions []PDRegion, storeRegionIdx map[uint64]*pdRegionSet, src, dst uint64) (uint64, bool) {
	srcSet := storeRegionIdx[src]
	if srcSet == nil {
		return 0, false
	}

	type candidate struct {
		regionID uint64
		isLeader bool
		size     int64
	}
	var candidates []candidate

	for _, region := range regions {
		if !srcSet.has(region.ID) {
			continue
		}
		hasDst := false
		for _, peer := range region.Peers {
			if peer.StoreID == dst {
				hasDst = true
				break
			}
		}
		if hasDst {
			continue
		}
		isLeader := region.Leader != nil && region.Leader.StoreID == src
		candidates = append(candidates, candidate{regionID: region.ID, isLeader: isLeader, size: region.ApproximateSize})
	}
	if len(candidates) == 0 {
		return 0, false
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].isLeader != candidates[j].isLeader {
			return !candidates[i].isLeader
		}
		return candidates[i].size < candidates[j].size
	})
	return candidates[0].regionID, true
}

func findPDRegionToTransferLeader(regions []PDRegion, src, dst uint64) (uint64, bool) {
	for _, region := range regions {
		if region.Leader == nil || region.Leader.StoreID != src {
			continue
		}
		for _, peer := range region.Peers {
			if peer.StoreID == dst {
				return region.ID, true
			}
		}
	}
	return 0, false
}

func transferPDPeerInRegions(regions []PDRegion, regionID, fromStore, toStore uint64) {
	for i := range regions {
		if regions[i].ID != regionID {
			continue
		}
		for j := range regions[i].Peers {
			if regions[i].Peers[j].StoreID == fromStore {
				regions[i].Peers[j].StoreID = toStore
				regions[i].Peers[j].ID = 0
				break
			}
		}
		if regions[i].Leader != nil && regions[i].Leader.StoreID == fromStore {
			regions[i].Leader.StoreID = toStore
			regions[i].Leader.ID = 0
		}
		break
	}
}

func transferPDLeaderInRegions(regions []PDRegion, regionID, toStore uint64) {
	for i := range regions {
		if regions[i].ID != regionID {
			continue
		}
		for j := range regions[i].Peers {
			if regions[i].Peers[j].StoreID == toStore {
				regions[i].Leader = &PDPeer{
					ID:       regions[i].Peers[j].ID,
					StoreID:  toStore,
					RoleName: "Voter",
				}
				break
			}
		}
		break
	}
}

func deepCopyPDRegions(regions []PDRegion) []PDRegion {
	cp := make([]PDRegion, len(regions))
	for i, r := range regions {
		cp[i] = PDRegion{
			ID:              r.ID,
			StartKey:        r.StartKey,
			EndKey:          r.EndKey,
			ApproximateSize: r.ApproximateSize,
		}
		cp[i].Peers = make([]PDPeer, len(r.Peers))
		copy(cp[i].Peers, r.Peers)
		if r.Leader != nil {
			leaderCopy := *r.Leader
			cp[i].Leader = &leaderCopy
		}
	}
	return cp
}

// --- Region set helper ---

type pdRegionSet struct {
	m map[uint64]struct{}
}

func newPDRegionSet() *pdRegionSet {
	return &pdRegionSet{m: make(map[uint64]struct{})}
}

func (s *pdRegionSet) add(id uint64)    { s.m[id] = struct{}{} }
func (s *pdRegionSet) remove(id uint64) { delete(s.m, id) }
func (s *pdRegionSet) has(id uint64) bool {
	_, ok := s.m[id]
	return ok
}

func buildPDStoreRegionIndex(regions []PDRegion) map[uint64]*pdRegionSet {
	index := make(map[uint64]*pdRegionSet)
	for _, region := range regions {
		for _, peer := range region.Peers {
			rs, ok := index[peer.StoreID]
			if !ok {
				rs = newPDRegionSet()
				index[peer.StoreID] = rs
			}
			rs.add(region.ID)
		}
	}
	return index
}

// --- Command execution helper ---

func runTiUPCommand(ctx context.Context, cmdStr string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// stderr from tiup often contains non-error progress info; include stdout too
		return stdout.String() + stderr.String(), err
	}
	return stdout.String(), nil
}

func balancerTruncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractJSON finds the first JSON object in the string.
// tiup ctl may print banner/info text before the actual JSON output.
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	// Find the matching closing brace by counting depth
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	// If no matching brace found, return from first '{' to end
	return s[start:]
}
