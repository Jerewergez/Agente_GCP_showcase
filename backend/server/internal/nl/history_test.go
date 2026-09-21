package nl

import (
	"testing"
	"time"
)

func TestNewHistoryStore(t *testing.T) {
	hs := NewHistoryStore(nil, HistoryConfig{
		MaxEntriesPerUser: 50,
	})
	if hs == nil {
		t.Fatal("expected non-nil HistoryStore")
	}
}

func TestHistoryStoreDefaults(t *testing.T) {
	hs := NewHistoryStore(nil, HistoryConfig{})
	if hs.cfg.MaxEntriesPerUser != 100 {
		t.Fatalf("expected default 100 max entries, got %d", hs.cfg.MaxEntriesPerUser)
	}
	if hs.cfg.FlushInterval != 30*time.Second {
		t.Fatalf("expected default 30s flush interval, got %v", hs.cfg.FlushInterval)
	}
}

func TestHistoryRecordAndGet(t *testing.T) {
	hs := NewHistoryStore(nil, HistoryConfig{MaxEntriesPerUser: 10})

	entry := &HistoryEntry{
		ID:        "qry_1",
		UserID:    "user@test.com",
		Query:     "show me FCR",
		SQL:       "SELECT ...",
		Source:    "template",
		Success:   true,
		LatencyMs: 150,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	hs.Record(entry)

	// Get history for user
	entries := hs.GetHistory("user@test.com", 0, 10)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Query != "show me FCR" {
		t.Fatalf("expected query 'show me FCR', got %q", entries[0].Query)
	}
}

func TestHistoryRecord_AnonymousUser(t *testing.T) {
	hs := NewHistoryStore(nil, HistoryConfig{MaxEntriesPerUser: 10})

	entry := &HistoryEntry{
		ID:    "qry_2",
		Query: "NPS last month",
		SQL:   "SELECT ...",
	}
	hs.Record(entry)

	entries := hs.GetHistory("", 0, 10)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for anonymous, got %d", len(entries))
	}
}

func TestHistoryRecord_MaxEntries(t *testing.T) {
	hs := NewHistoryStore(nil, HistoryConfig{MaxEntriesPerUser: 3})

	for i := 0; i < 10; i++ {
		hs.Record(&HistoryEntry{
			ID:    "qry",
			Query: "query",
		})
	}

	entries := hs.GetHistory("__anonymous__", 0, 100)
	if len(entries) > 3 {
		t.Fatalf("expected max 3 entries, got %d", len(entries))
	}
}

func TestHistoryGet_OffsetLimit(t *testing.T) {
	hs := NewHistoryStore(nil, HistoryConfig{MaxEntriesPerUser: 100})

	// Add 5 entries (newest first)
	for i := 5; i >= 1; i-- {
		hs.Record(&HistoryEntry{
			ID:    "",
			Query: "query",
		})
	}

	// Get with offset
	entries := hs.GetHistory("__anonymous__", 2, 2)
	if len(entries) > 2 {
		t.Fatalf("expected max 2 entries with limit, got %d", len(entries))
	}
}

func TestHistoryGet_BeyondLength(t *testing.T) {
	hs := NewHistoryStore(nil, HistoryConfig{MaxEntriesPerUser: 10})
	hs.Record(&HistoryEntry{ID: "qry", Query: "test"})

	entries := hs.GetHistory("__anonymous__", 100, 10)
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries for out-of-range offset, got %d", len(entries))
	}
}

func TestHistoryRecord_NilEntry(t *testing.T) {
	hs := NewHistoryStore(nil, HistoryConfig{MaxEntriesPerUser: 10})
	hs.Record(nil) // Should not panic

	entries := hs.GetHistory("__anonymous__", 0, 10)
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries after nil record, got %d", len(entries))
	}
}

func TestHistoryStats(t *testing.T) {
	hs := NewHistoryStore(nil, HistoryConfig{MaxEntriesPerUser: 50})

	hs.Record(&HistoryEntry{ID: "q1", Query: "q1"})
	hs.Record(&HistoryEntry{ID: "q2", Query: "q2"})

	stats := hs.Stats()
	if stats["users"].(int) != 1 {
		t.Fatalf("expected 1 user, got %d", stats["users"])
	}
	if stats["max_per_user"].(int) != 50 {
		t.Fatalf("expected max 50 per user, got %d", stats["max_per_user"])
	}
}

func TestNewHistoryEntry(t *testing.T) {
	entry := NewHistoryEntry("user@test.com", "show me FCR", "SELECT 1", "template", true, "", 100)
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.UserID != "user@test.com" {
		t.Fatalf("expected user_id 'user@test.com', got %q", entry.UserID)
	}
	if entry.Query != "show me FCR" {
		t.Fatalf("expected query 'show me FCR', got %q", entry.Query)
	}
	if entry.Source != "template" {
		t.Fatalf("expected source 'template', got %q", entry.Source)
	}
	if !entry.Success {
		t.Fatal("expected success true")
	}
	if entry.LatencyMs != 100 {
		t.Fatalf("expected latency 100ms, got %d", entry.LatencyMs)
	}
	if entry.CreatedAt == "" {
		t.Fatal("expected non-empty created_at")
	}
}

func TestNewHistoryEntry_WithError(t *testing.T) {
	entry := NewHistoryEntry("user@test.com", "bad query", "", "gemma", false, "parse error", 50)
	if entry.Success {
		t.Fatal("expected success false")
	}
	if entry.ErrorMsg != "parse error" {
		t.Fatalf("expected error 'parse error', got %q", entry.ErrorMsg)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := GenerateID()
	id2 := GenerateID()
	if id1 == id2 {
		t.Fatal("expected unique IDs")
	}
	if len(id1) < 10 {
		t.Fatalf("expected ID length >= 10, got %d", len(id1))
	}
}

func TestParseHistoryRequest(t *testing.T) {
	req, err := ParseHistoryRequest("user@test.com", "0", "50")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.UserID != "user@test.com" {
		t.Fatalf("expected user_id 'user@test.com', got %q", req.UserID)
	}
	if req.Offset != 0 {
		t.Fatalf("expected offset 0, got %d", req.Offset)
	}
	if req.Limit != 50 {
		t.Fatalf("expected limit 50, got %d", req.Limit)
	}
}

func TestParseHistoryRequest_InvalidOffset(t *testing.T) {
	_, err := ParseHistoryRequest("user", "abc", "50")
	if err == nil {
		t.Fatal("expected error for invalid offset")
	}
}

func TestParseHistoryRequest_DefaultLimit(t *testing.T) {
	req, err := ParseHistoryRequest("user", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Limit != 50 {
		t.Fatalf("expected default limit 50, got %d", req.Limit)
	}
}

func TestParseHistoryRequest_ClampLimit(t *testing.T) {
	req, err := ParseHistoryRequest("user", "0", "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Limit != 50 {
		t.Fatalf("expected clamped limit 50, got %d", req.Limit)
	}
}

func TestParseHistoryRequest_NegativeOffset(t *testing.T) {
	req, err := ParseHistoryRequest("user", "-5", "10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Offset != 0 {
		t.Fatalf("expected offset clamped to 0, got %d", req.Offset)
	}
}
