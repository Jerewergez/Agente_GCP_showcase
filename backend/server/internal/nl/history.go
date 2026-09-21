package nl

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Jerewergez/Agente_Workspace/internal/bq"
)

// ---------------------------------------------------------------------------
// QueryHistory — per-user in-memory query history with async BQ persistence
// ---------------------------------------------------------------------------

// HistoryEntry represents a single query in the history log.
type HistoryEntry struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Query     string `json:"query"`
	SQL       string `json:"sql,omitempty"`
	Source    string `json:"source"` // "template" or "gemma"
	Success   bool   `json:"success"`
	ErrorMsg  string `json:"error,omitempty"`
	LatencyMs int64  `json:"latency_ms"`
	CreatedAt string `json:"created_at"`
}

// HistoryConfig holds configuration for the history store.
type HistoryConfig struct {
	// MaxEntriesPerUser limits in-memory entries per user (default 100).
	MaxEntriesPerUser int
	// BQTable is the BigQuery table for persistent storage.
	// Expected format: project.dataset.agente_memory_interactions
	BQTable string
	// FlushInterval is how often to flush buffered entries to BQ.
	FlushInterval time.Duration
}

// HistoryStore manages query history in memory and persists to BigQuery.
type HistoryStore struct {
	bqClient *bq.Client
	cfg      HistoryConfig

	mu        sync.RWMutex
	entries   map[string][]*HistoryEntry // userID → history entries (newest first)
	buffer    []*HistoryEntry            // entries pending BQ insert
	totalKept int

	stopCh chan struct{}
}

// NewHistoryStore creates a new query history store.
func NewHistoryStore(bqClient *bq.Client, cfg HistoryConfig) *HistoryStore {
	if cfg.MaxEntriesPerUser <= 0 {
		cfg.MaxEntriesPerUser = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 30 * time.Second
	}

	hs := &HistoryStore{
		bqClient: bqClient,
		cfg:      cfg,
		entries:  make(map[string][]*HistoryEntry),
		buffer:   make([]*HistoryEntry, 0, 100),
		stopCh:   make(chan struct{}),
	}

	if bqClient != nil && cfg.BQTable != "" {
		go hs.flushLoop()
	}

	return hs
}

// Stop halts the background flush loop.
func (hs *HistoryStore) Stop() {
	close(hs.stopCh)
}

// Record adds an entry to the in-memory history and enqueues it for
// async BigQuery persistence. It returns immediately (non-blocking).
func (hs *HistoryStore) Record(entry *HistoryEntry) {
	if entry == nil {
		return
	}
	if entry.CreatedAt == "" {
		entry.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	hs.mu.Lock()
	defer hs.mu.Unlock()

	// Prepend to user's in-memory history
	userID := entry.UserID
	if userID == "" {
		userID = "__anonymous__"
	}
	entry.UserID = userID

	entries := hs.entries[userID]
	entries = append([]*HistoryEntry{entry}, entries...) // prepend

	// Trim to max
	if len(entries) > hs.cfg.MaxEntriesPerUser {
		entries = entries[:hs.cfg.MaxEntriesPerUser]
	}
	hs.entries[userID] = entries
	hs.totalKept++

	// Buffer for BQ insert
	if hs.bqClient != nil && hs.cfg.BQTable != "" {
		hs.buffer = append(hs.buffer, entry)
	}
}

// GetHistory returns the last N query history entries for a user.
// Results are newest-first.
func (hs *HistoryStore) GetHistory(userID string, offset, limit int) []*HistoryEntry {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	if userID == "" {
		userID = "__anonymous__"
	}

	entries := hs.entries[userID]
	if entries == nil {
		return []*HistoryEntry{}
	}

	if offset >= len(entries) {
		return []*HistoryEntry{}
	}

	end := offset + limit
	if end > len(entries) || limit <= 0 {
		end = len(entries)
	}

	result := make([]*HistoryEntry, end-offset)
	copy(result, entries[offset:end])
	return result
}

// GetStats returns statistics about the history store.
func (hs *HistoryStore) Stats() map[string]interface{} {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	return map[string]interface{}{
		"users":       len(hs.entries),
		"total_kept":  hs.totalKept,
		"buffer_size": len(hs.buffer),
		"max_per_user": hs.cfg.MaxEntriesPerUser,
	}
}

// flushLoop periodically flushes buffered entries to BigQuery.
func (hs *HistoryStore) flushLoop() {
	ticker := time.NewTicker(hs.cfg.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hs.flushToBQ()
		case <-hs.stopCh:
			// Final flush before exiting
			hs.flushToBQ()
			return
		}
	}
}

// flushToBQ writes buffered entries to BigQuery asynchronously.
func (hs *HistoryStore) flushToBQ() {
	hs.mu.Lock()
	if len(hs.buffer) == 0 {
		hs.mu.Unlock()
		return
	}
	batch := make([]*HistoryEntry, len(hs.buffer))
	copy(batch, hs.buffer)
	hs.buffer = hs.buffer[:0]
	hs.mu.Unlock()

	// Non-blocking async insert
	go func(entries []*HistoryEntry) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := hs.insertBatch(ctx, entries); err != nil {
			log.Printf("nl/history: bq insert batch failed (%d entries): %v", len(entries), err)
		} else {
			log.Printf("nl/history: flushed %d entries to BQ", len(entries))
		}
	}(batch)
}

// insertBatch inserts a batch of history entries into BigQuery.
// It uses a streaming insert via the BigQuery client.
func (hs *HistoryStore) insertBatch(ctx context.Context, entries []*HistoryEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Parse the BQ table reference
	tableRef := hs.cfg.BQTable
	if !strings.Contains(tableRef, ".") {
		return fmt.Errorf("nl/history: BQTable %q must be project.dataset.table format", tableRef)
	}

	// Build JSON rows for streaming insert
	var rows []interface{}
	for _, e := range entries {
		// Convert to a map for BigQuery struct insertion
		row := map[string]interface{}{
			"id":         e.ID,
			"user_id":    e.UserID,
			"query":      e.Query,
			"sql":        e.SQL,
			"source":     e.Source,
			"success":    e.Success,
			"error_msg":  e.ErrorMsg,
			"latency_ms": e.LatencyMs,
			"created_at": e.CreatedAt,
		}
		rows = append(rows, row)
	}

	return hs.insertRows(ctx, tableRef, rows)
}

// insertRows inserts rows using a multi-row INSERT statement.
func (hs *HistoryStore) insertRows(ctx context.Context, tableRef string, rows []interface{}) error {
	if len(rows) == 0 {
		return nil
	}

	var valueClauses []string
	for _, row := range rows {
		r, ok := row.(map[string]interface{})
		if !ok {
			continue
		}
		vals := fmt.Sprintf(
			"('%s', '%s', '%s', '%s', '%s', %t, '%s', %d, '%s')",
			escapeBQString(fmt.Sprintf("%v", r["id"])),
			escapeBQString(fmt.Sprintf("%v", r["user_id"])),
			escapeBQString(fmt.Sprintf("%v", r["query"])),
			escapeBQString(fmt.Sprintf("%v", r["sql"])),
			escapeBQString(fmt.Sprintf("%v", r["source"])),
			fmt.Sprintf("%v", r["success"]) == "true",
			escapeBQString(fmt.Sprintf("%v", r["error_msg"])),
			toInt64(r["latency_ms"]),
			escapeBQString(fmt.Sprintf("%v", r["created_at"])),
		)
		valueClauses = append(valueClauses, vals)
	}

	if len(valueClauses) == 0 {
		return nil
	}

	insertSQL := fmt.Sprintf(
		`INSERT INTO %s (id, user_id, query, sql, source, success, error_msg, latency_ms, created_at) VALUES %s`,
		tableRef,
		strings.Join(valueClauses, ", "),
	)

	qb := bq.NewQuery(insertSQL)
	_, err := hs.bqClient.Run(ctx, qb)
	if err != nil {
		return fmt.Errorf("insert history rows: %w", err)
	}
	return nil
}

func escapeBQString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case float64:
		return int64(n)
	case int:
		return int64(n)
	default:
		return 0
	}
}

// GenerateID creates a simple unique query ID.
func GenerateID() string {
	return fmt.Sprintf("qry_%d", time.Now().UnixNano())
}

// NewHistoryEntry creates a new HistoryEntry with the given parameters.
func NewHistoryEntry(userID, query, sql, source string, success bool, errMsg string, latencyMs int64) *HistoryEntry {
	return &HistoryEntry{
		ID:        GenerateID(),
		UserID:    userID,
		Query:     query,
		SQL:       sql,
		Source:    source,
		Success:   success,
		ErrorMsg:  errMsg,
		LatencyMs: latencyMs,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// ---------------------------------------------------------------------------
// HTTP handler helpers for query history endpoints
// ---------------------------------------------------------------------------

// QueryHistoryRequest holds the parsed query parameters for history lookup.
type QueryHistoryRequest struct {
	UserID string `json:"user_id"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

// ParseHistoryRequest parses query parameters into a request struct.
func ParseHistoryRequest(userID, offsetStr, limitStr string) (*QueryHistoryRequest, error) {
	offset := 0
	limit := 50

	if offsetStr != "" {
		if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil {
			return nil, fmt.Errorf("invalid offset: %w", err)
		}
		if offset < 0 {
			offset = 0
		}
	}

	if limitStr != "" {
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
			return nil, fmt.Errorf("invalid limit: %w", err)
		}
		if limit <= 0 || limit > 200 {
			limit = 50
		}
	}

	return &QueryHistoryRequest{
		UserID: userID,
		Offset: offset,
		Limit:  limit,
	}, nil
}

// HistoryResponse is the JSON response for history queries.
type HistoryResponse struct {
	Entries  []*HistoryEntry `json:"entries"`
	Total    int             `json:"total"`
	Offset   int             `json:"offset"`
	Limit    int             `json:"limit"`
	UserID   string          `json:"user_id"`
}

// GetHistoryResponse builds a paginated history response.
func (hs *HistoryStore) GetHistoryResponse(req *QueryHistoryRequest) *HistoryResponse {
	entries := hs.GetHistory(req.UserID, req.Offset, req.Limit)
	return &HistoryResponse{
		Entries: entries,
		Total:   len(hs.entries[req.UserID]),
		Offset:  req.Offset,
		Limit:   req.Limit,
		UserID:  req.UserID,
	}
}

// Internal helper to get total entry count for a user.
func (hs *HistoryStore) totalEntries(userID string) int {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return len(hs.entries[userID])
}
