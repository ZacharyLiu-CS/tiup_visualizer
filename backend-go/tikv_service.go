package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	kv2graph "git.woa.com/zacharyzliu/KV2Graph/pkg"

	pb "git.woa.com/zacharyzliu/KV2Graph/protos"
)

const (
	tikvClientTTL    = 10 * time.Minute
	tikvClientPrune  = 1 * time.Minute
	defaultScanLimit = 1000
)

// tikvClientEntry holds a TiKV client with its last access time.
type tikvClientEntry struct {
	client    *kv2graph.TiKVClient
	pdAddr    string
	cf        string
	lastUsed  time.Time
}

// TiKVService manages TiKV client connections and provides data access methods.
type TiKVService struct {
	mu      sync.Mutex
	clients map[string]*tikvClientEntry // key: "pdAddr|cf"
	stopCh  chan struct{}
}

// NewTiKVService creates a new TiKVService with background client pruning.
func NewTiKVService() *TiKVService {
	s := &TiKVService{
		clients: make(map[string]*tikvClientEntry),
		stopCh:  make(chan struct{}),
	}
	go s.pruneLoop()
	return s
}

// Close stops the background prune loop and closes all clients.
func (s *TiKVService) Close() {
	close(s.stopCh)
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, entry := range s.clients {
		slog.Info("Closing TiKV client", "pd", entry.pdAddr, "cf", entry.cf)
		entry.client.Close()
		delete(s.clients, key)
	}
}

func clientKey(pdAddr, cf string) string {
	return pdAddr + "|" + cf
}

func (s *TiKVService) getClient(pdAddr, cf string) (*kv2graph.TiKVClient, error) {
	key := clientKey(pdAddr, cf)
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.clients[key]; ok {
		entry.lastUsed = time.Now()
		return entry.client, nil
	}

	client, err := kv2graph.NewTiKVClient(pdAddr, cf)
	if err != nil {
		return nil, err
	}

	s.clients[key] = &tikvClientEntry{
		client:   client,
		pdAddr:   pdAddr,
		cf:       cf,
		lastUsed: time.Now(),
	}
	slog.Info("Created TiKV client", "pd", pdAddr, "cf", cf, "totalClients", len(s.clients))
	return client, nil
}

func (s *TiKVService) pruneLoop() {
	ticker := time.NewTicker(tikvClientPrune)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.pruneIdleClients()
		}
	}
}

func (s *TiKVService) pruneIdleClients() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for key, entry := range s.clients {
		if now.Sub(entry.lastUsed) > tikvClientTTL {
			slog.Info("Pruning idle TiKV client", "pd", entry.pdAddr, "cf", entry.cf)
			entry.client.Close()
			delete(s.clients, key)
		}
	}
}

// --- Query Methods ---

// GetKey retrieves a single key from TiKV and returns structured result.
func (s *TiKVService) GetKey(pdAddr, cf, key string, parseType string) (interface{}, error) {
	client, err := s.getClient(pdAddr, cf)
	if err != nil {
		return nil, fmt.Errorf("connect TiKV failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	val, err := client.Get(ctx, []byte(key))
	if err != nil {
		return nil, fmt.Errorf("get key failed: %w", err)
	}
	if val == nil {
		return nil, fmt.Errorf("key %q not found", key)
	}

	switch parseType {
	case "hex":
		return kv2graph.RawValueToJSON(key, val), nil
	case "graph_meta":
		meta, err := kv2graph.ParseGraphMeta(val)
		if err != nil {
			return nil, fmt.Errorf("parse GraphMeta failed: %w", err)
		}
		return kv2graph.GraphMetaMap(meta)
	default:
		return nil, fmt.Errorf("unsupported parse-type: %s", parseType)
	}
}

// ScanPrefix scans keys with the given prefix from TiKV and returns structured results.
func (s *TiKVService) ScanPrefix(pdAddr, cf, prefix string, limit int, parseType string) ([]interface{}, error) {
	client, err := s.getClient(pdAddr, cf)
	if err != nil {
		return nil, fmt.Errorf("connect TiKV failed: %w", err)
	}

	if limit <= 0 {
		limit = defaultScanLimit
	}
	endKey := prefix + "~"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	keys, values, err := client.Scan(ctx, []byte(prefix), []byte(endKey), limit)
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	var results []interface{}
	for i := range keys {
		keyStr := string(keys[i])
		switch parseType {
		case "hex":
			results = append(results, kv2graph.RawValueToJSON(keyStr, values[i]))
		case "graph_meta":
			meta, err := kv2graph.ParseGraphMeta(values[i])
			if err != nil {
				slog.Warn("Failed to parse GraphMeta", "key", keyStr, "error", err)
				// Return hex fallback for unparseable entries
				item := kv2graph.RawValueToJSON(keyStr, values[i])
				item["parseError"] = err.Error()
				results = append(results, item)
				continue
			}
			m, err := kv2graph.GraphMetaMap(meta)
			if err != nil {
				slog.Warn("Failed to serialize GraphMeta", "key", keyStr, "error", err)
				results = append(results, kv2graph.RawValueToJSON(keyStr, values[i]))
				continue
			}
			m["key"] = keyStr
			results = append(results, m)
		}
	}
	return results, nil
}

// ExtractPDAddrs extracts PD addresses from cluster components.
// Returns a comma-separated string of PD host:port pairs.
// Uses comp.ID (e.g. "9.135.148.20:22379") which includes the port,
// not comp.Host (e.g. "9.135.148.20") which is IP only.
func ExtractPDAddrs(detail *ClusterDetail) string {
	var addrs []string
	for _, comp := range detail.Components {
		if comp.Role == "pd" {
			addrs = append(addrs, comp.ID)
		}
	}
	return joinStrings(addrs, ",")
}

// helper to join non-empty strings (avoid importing strings just for this in tests)
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, p := range parts[1:] {
		if p != "" {
			result += sep + p
		}
	}
	return result
}

// Ensure pb and kv2graph are used (compile check).
var _ *pb.GraphMeta
var _ = kv2graph.ParseGraphMeta
