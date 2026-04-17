package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"region-balancer/pkg"
)

// --- Balance operation types (API-facing, kept for frontend compat) ---

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

// --- SSE per-operation status types ---

type OpJSON struct {
	Type      string `json:"type"`
	RegionID  uint64 `json:"region_id"`
	FromStore uint64 `json:"from_store,omitempty"`
	ToStore   uint64 `json:"to_store"`
	Reason    string `json:"reason,omitempty"`
}

type BatchOpStatus struct {
	Operation   OpJSON     `json:"operation"`
	Status      string     `json:"status"` // "submitted", "in_progress", "completed", "failed"
	CurrentStep int        `json:"current_step,omitempty"`
	TotalSteps  int        `json:"total_steps,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	Elapsed     string     `json:"elapsed,omitempty"`
	Error       string     `json:"error,omitempty"`
}

type BalanceTask struct {
	ID              string         `json:"id"`
	Status          TaskStatus     `json:"status"`
	Config          TaskConfig     `json:"config"`
	Plan            *AnalyzeResult `json:"plan,omitempty"`
	Progress        int            `json:"progress"`
	Total           int            `json:"total"`
	Results         []OpResult     `json:"results"`
	Error           string         `json:"error,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	FinishedAt      *time.Time     `json:"finished_at,omitempty"`
	CurrentBatch    int            `json:"current_batch"`
	TotalBatches    int            `json:"total_batches"`
	BatchOperations []BatchOpStatus `json:"batch_operations"`

	cancel context.CancelFunc `json:"-"`
}

type OpResult struct {
	Operation BalanceOperation `json:"operation"`
	Status    string           `json:"status"` // "success", "failed", "skipped"
	Error     string           `json:"error,omitempty"`
	Duration  string           `json:"duration,omitempty"`
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

// --- Task Execution (custom loop with per-op SSE) ---

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

	// Create TiUPClient from config
	client, err := pkg.NewTiUPClient(config.PDAddr, config.TiUPVersion, 60*time.Second)
	if err != nil {
		s.mu.Lock()
		task.Status = TaskFailed
		task.Error = fmt.Sprintf("failed to create TiUP client: %v", err)
		finishTime := time.Now()
		task.FinishedAt = &finishTime
		task.UpdatedAt = finishTime
		s.mu.Unlock()
		s.broadcast(SSEEvent{Type: "task_update", Data: task})
		slog.Error("Balancer: failed to create TiUP client", "task_id", taskID, "error", err)
		return
	}

	slog.Info("Balancer: fetching region data", "task_id", taskID, "pd_addr", config.PDAddr)

	// Convert BalanceOperations to pkg.Operations for batching
	pkgOps := make([]pkg.Operation, len(operations))
	for i, op := range operations {
		pkgOps[i] = balanceOpToPkgOp(op)
	}

	// Split into batches
	batches := splitBalanceOpBatches(operations, batchSize)
	pkgBatches := splitPkgOpBatches(pkgOps, batchSize)
	totalBatches := len(batches)

	s.mu.Lock()
	task.TotalBatches = totalBatches
	s.mu.Unlock()

	for batchIdx := 0; batchIdx < totalBatches; batchIdx++ {
		// Check cancellation
		select {
		case <-ctx.Done():
			slog.Warn("Execution cancelled, finishing current progress", "task_id", taskID)
			s.markTaskCancelled(task)
			return
		default:
		}

		batch := batches[batchIdx]
		pkgBatch := pkgBatches[batchIdx]
		batchNum := batchIdx + 1

		slog.Info(fmt.Sprintf("Executing batch %d/%d (%d operations)", batchNum, totalBatches, len(batch)), "task_id", taskID)

		// Build initial BatchOpStatus for this batch
		batchStatuses := make([]BatchOpStatus, len(batch))
		for i, op := range batch {
			batchStatuses[i] = BatchOpStatus{
				Operation: balanceOpToOpJSON(op),
				Status:    "submitted",
			}
		}

		s.mu.Lock()
		task.CurrentBatch = batchNum
		task.BatchOperations = batchStatuses
		task.UpdatedAt = time.Now()
		s.mu.Unlock()

		// Step a: Submit each operation
		for i, pkgOp := range pkgBatch {
			select {
			case <-ctx.Done():
				slog.Warn("Execution cancelled, finishing current progress", "task_id", taskID)
				s.markTaskCancelled(task)
				return
			default:
			}

			submitErr := submitPkgOperation(ctx, client, pkgOp)
			startTime := time.Now()

			s.mu.Lock()
			if submitErr != nil {
				batchStatuses[i].Status = "failed"
				batchStatuses[i].Error = submitErr.Error()
				slog.Error(fmt.Sprintf("  Failed to submit: %s", pkgOp.String()), "task_id", taskID, "error", submitErr)

				task.Results = append(task.Results, OpResult{
					Operation: batch[i],
					Status:    "failed",
					Error:     submitErr.Error(),
				})
				task.Progress = len(task.Results)
			} else {
				batchStatuses[i].Status = "submitted"
				batchStatuses[i].StartedAt = &startTime
				slog.Info(fmt.Sprintf("  Submitted: %s", pkgOp.String()), "task_id", taskID)
			}
			task.BatchOperations = batchStatuses
			task.UpdatedAt = time.Now()
			s.mu.Unlock()

			s.broadcast(SSEEvent{Type: "task_update", Data: task})
		}

		// Step c: Poll via operator show every 5 seconds until all complete
		pendingByRegion := make(map[uint64]int) // regionID -> index in batch
		for i, op := range pkgBatch {
			if batchStatuses[i].Status == "submitted" {
				pendingByRegion[op.RegionID] = i
			}
		}

		ticker := time.NewTicker(5 * time.Second)
		for len(pendingByRegion) > 0 {
			select {
			case <-ctx.Done():
				ticker.Stop()
				slog.Warn("Execution cancelled, finishing current progress", "task_id", taskID)
				s.markTaskCancelled(task)
				return
			case <-ticker.C:
			}

			activeOps, pollErr := client.ShowOperatorsParsed(ctx)
			if pollErr != nil {
				slog.Warn("operator show failed, will retry", "task_id", taskID, "error", pollErr)
				continue
			}

			// Build active region map
			activeRegions := make(map[uint64]*pkg.OperatorStatus)
			for idx := range activeOps {
				activeRegions[activeOps[idx].RegionID] = &activeOps[idx]
			}

			// Check each pending op
			for regionID, batchIdx := range pendingByRegion {
				if status, active := activeRegions[regionID]; active {
					// Still running - update progress
					s.mu.Lock()
					batchStatuses[batchIdx].Status = "in_progress"
					batchStatuses[batchIdx].CurrentStep = status.CurrentStep
					batchStatuses[batchIdx].TotalSteps = status.TotalSteps
					if batchStatuses[batchIdx].StartedAt != nil {
						batchStatuses[batchIdx].Elapsed = time.Since(*batchStatuses[batchIdx].StartedAt).Round(time.Millisecond).String()
					}
					task.BatchOperations = batchStatuses
					task.UpdatedAt = time.Now()
					s.mu.Unlock()

					slog.Info(fmt.Sprintf("  In progress: %s (step %d/%d)", pkgBatch[batchIdx].String(), status.CurrentStep, status.TotalSteps), "task_id", taskID)
				} else {
					// No longer in operator show -> completed
					s.mu.Lock()
					batchStatuses[batchIdx].Status = "completed"
					if batchStatuses[batchIdx].StartedAt != nil {
						batchStatuses[batchIdx].Elapsed = time.Since(*batchStatuses[batchIdx].StartedAt).Round(time.Millisecond).String()
					}
					task.BatchOperations = batchStatuses
					task.Results = append(task.Results, OpResult{
						Operation: batch[batchIdx],
						Status:    "success",
						Duration:  batchStatuses[batchIdx].Elapsed,
					})
					task.Progress = len(task.Results)
					task.UpdatedAt = time.Now()
					s.mu.Unlock()

					slog.Info(fmt.Sprintf("  Completed: %s", pkgBatch[batchIdx].String()), "task_id", taskID)
					delete(pendingByRegion, regionID)
				}
			}

			s.broadcast(SSEEvent{Type: "task_update", Data: task})
		}
		ticker.Stop()

		// Batch summary
		succeeded := 0
		failed := 0
		for _, bs := range batchStatuses {
			if bs.Status == "completed" {
				succeeded++
			} else if bs.Status == "failed" {
				failed++
			}
		}
		slog.Info(fmt.Sprintf("Batch %d/%d complete: %d succeeded, %d failed", batchNum, totalBatches, succeeded, failed), "task_id", taskID)
	}

	// Finalize
	s.mu.Lock()
	finishTime := time.Now()
	task.FinishedAt = &finishTime
	task.UpdatedAt = finishTime
	task.Status = TaskCompleted
	// Check if any ops failed
	for _, r := range task.Results {
		if r.Status == "failed" {
			// Still completed, with some failures noted
			break
		}
	}
	task.BatchOperations = nil // clear batch operations on finish
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

// --- Analyze (dry-run) using pkg library ---

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

	slog.Info("Balancer: fetching region data", "pd_addr", pdAddr, "tiup_version", tiupVersion)

	// Create TiUPClient and fetcher
	client, err := pkg.NewTiUPClient(pdAddr, tiupVersion, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to create TiUP client: %w", err)
	}
	fetcher := pkg.NewTiUPFetcher(client)

	// Fetch regions
	resp, err := fetcher.FetchRegions(context.Background())
	if err != nil {
		slog.Error("Balancer: failed to fetch regions", "error", err)
		return nil, fmt.Errorf("failed to fetch regions: %w", err)
	}
	if len(resp.Regions) == 0 {
		return nil, fmt.Errorf("no regions found")
	}

	// Plan using pkg.Planner
	planner := pkg.NewPlanner(pkg.PlannerConfig{
		PeerThreshold:   peerThreshold,
		LeaderThreshold: leaderThreshold,
	})
	plan, err := planner.Plan(resp.Regions)
	if err != nil {
		return nil, fmt.Errorf("failed to plan: %w", err)
	}

	// Also get distribution report for before/after stats
	beforeReport := pkg.AnalyzeDistribution(resp.Regions)

	// Convert to AnalyzeResult for frontend compatibility
	return convertToAnalyzeResult(plan, beforeReport), nil
}

// --- Conversion helpers ---

// convertToAnalyzeResult converts pkg types to the frontend-compatible AnalyzeResult.
func convertToAnalyzeResult(plan *pkg.BalancePlan, beforeReport *pkg.DistributionReport) *AnalyzeResult {
	numStores := beforeReport.TotalStores
	idealPeers := beforeReport.IdealPeers
	idealLeaders := beforeReport.IdealLeaders

	// Build before distribution
	beforeDist := convertStoreStatsToDist(plan.BeforeStats, idealPeers, idealLeaders)

	// Build after distribution
	afterDist := convertStoreStatsToDist(plan.AfterStats, idealPeers, idealLeaders)

	// Convert operations
	var ops []BalanceOperation
	peerOps := 0
	leaderOps := 0
	for _, op := range plan.Operations {
		bop := pkgOpToBalanceOp(op)
		ops = append(ops, bop)
		switch op.Type {
		case pkg.TransferPeer:
			peerOps++
		case pkg.TransferLeader:
			leaderOps++
		}
	}
	if ops == nil {
		ops = []BalanceOperation{}
	}

	return &AnalyzeResult{
		TotalRegions: beforeReport.TotalRegions,
		TotalStores:  numStores,
		TotalPeers:   beforeReport.TotalPeers,
		TotalLeaders: beforeReport.TotalLeaders,
		IdealPeers:   math.Round(idealPeers*100) / 100,
		IdealLeaders: math.Round(idealLeaders*100) / 100,
		Before:       beforeDist,
		After:        afterDist,
		Operations:   ops,
		PeerOps:      peerOps,
		LeaderOps:    leaderOps,
	}
}

func convertStoreStatsToDist(stats map[uint64]*pkg.StoreStats, idealPeers, idealLeaders float64) []StoreDistribution {
	dist := make([]StoreDistribution, 0, len(stats))
	for _, st := range stats {
		dist = append(dist, StoreDistribution{
			StoreID:     st.StoreID,
			PeerCount:   st.PeerCount,
			LeaderCount: st.LeaderCount,
			PeerDelta:   math.Round((float64(st.PeerCount)-idealPeers)*100) / 100,
			LeaderDelta: math.Round((float64(st.LeaderCount)-idealLeaders)*100) / 100,
		})
	}
	sort.Slice(dist, func(i, j int) bool { return dist[i].StoreID < dist[j].StoreID })
	return dist
}

func pkgOpToBalanceOp(op pkg.Operation) BalanceOperation {
	var opType BalanceOpType
	switch op.Type {
	case pkg.TransferPeer:
		opType = OpTransferPeer
	case pkg.TransferLeader:
		opType = OpTransferLeader
	default:
		opType = BalanceOpType(op.Type.String())
	}
	return BalanceOperation{
		Type:      opType,
		RegionID:  op.RegionID,
		FromStore: op.FromStore,
		ToStore:   op.ToStore,
		Reason:    op.Reason,
	}
}

func balanceOpToPkgOp(op BalanceOperation) pkg.Operation {
	var opType pkg.OperationType
	switch op.Type {
	case OpTransferPeer:
		opType = pkg.TransferPeer
	case OpTransferLeader:
		opType = pkg.TransferLeader
	}
	return pkg.Operation{
		Type:      opType,
		RegionID:  op.RegionID,
		FromStore: op.FromStore,
		ToStore:   op.ToStore,
		Reason:    op.Reason,
	}
}

func balanceOpToOpJSON(op BalanceOperation) OpJSON {
	return OpJSON{
		Type:      string(op.Type),
		RegionID:  op.RegionID,
		FromStore: op.FromStore,
		ToStore:   op.ToStore,
		Reason:    op.Reason,
	}
}

func submitPkgOperation(ctx context.Context, client *pkg.TiUPClient, op pkg.Operation) error {
	switch op.Type {
	case pkg.TransferPeer:
		_, err := client.RunOperator(ctx, "add", "transfer-peer",
			fmt.Sprintf("%d", op.RegionID),
			fmt.Sprintf("%d", op.FromStore),
			fmt.Sprintf("%d", op.ToStore))
		return err
	case pkg.TransferLeader:
		_, err := client.RunOperator(ctx, "add", "transfer-leader",
			fmt.Sprintf("%d", op.RegionID),
			fmt.Sprintf("%d", op.ToStore))
		return err
	default:
		return fmt.Errorf("unknown operation type: %v", op.Type)
	}
}

func splitBalanceOpBatches(ops []BalanceOperation, batchSize int) [][]BalanceOperation {
	if batchSize <= 0 {
		batchSize = 1
	}
	var batches [][]BalanceOperation
	for i := 0; i < len(ops); i += batchSize {
		end := i + batchSize
		if end > len(ops) {
			end = len(ops)
		}
		batches = append(batches, ops[i:end])
	}
	return batches
}

func splitPkgOpBatches(ops []pkg.Operation, batchSize int) [][]pkg.Operation {
	if batchSize <= 0 {
		batchSize = 1
	}
	var batches [][]pkg.Operation
	for i := 0; i < len(ops); i += batchSize {
		end := i + batchSize
		if end > len(ops) {
			end = len(ops)
		}
		batches = append(batches, ops[i:end])
	}
	return batches
}
