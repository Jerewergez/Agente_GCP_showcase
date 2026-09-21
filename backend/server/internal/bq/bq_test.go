package bq

import (
	"testing"
)

func TestNewQuery_Basic(t *testing.T) {
	qb := NewQuery("SELECT * FROM `project.dataset.table` WHERE id = ?")
	if qb == nil {
		t.Fatal("expected non-nil QueryBuilder")
	}

	sql, args := qb.SQL()
	if sql != "SELECT * FROM `project.dataset.table` WHERE id = ?" {
		t.Fatalf("unexpected SQL: %s", sql)
	}
	if len(args) != 0 {
		t.Fatalf("expected 0 args, got %d", len(args))
	}
}

func TestNewQuery_WithArgs(t *testing.T) {
	qb := NewQuery("SELECT * FROM t WHERE a = ? AND b = ?").
		WithArg("hello").
		WithArg(42)

	sql, args := qb.SQL()
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[0] != "hello" {
		t.Fatalf("expected arg[0]='hello', got %v", args[0])
	}
	if args[1] != 42 {
		t.Fatalf("expected arg[1]=42, got %v", args[1])
	}
	_ = sql // SQL is preserved
}

func TestNewQuery_WithArgsMultiple(t *testing.T) {
	qb := NewQuery("SELECT * FROM t WHERE x IN (?, ?, ?)").
		WithArgs("a", "b", "c")

	_, args := qb.SQL()
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(args))
	}
}

func TestCostEstimate_ZeroWhenCacheHit(t *testing.T) {
	ce := &CostEstimate{
		TotalBytesProcessed: 1_000_000_000,
		TotalBytesBilled:    1_000_000_000,
		CacheHit:            true,
		StatementType:       "SELECT",
	}

	if ce.EstimatedBytes() != 0 {
		t.Fatalf("expected 0 bytes for cache hit, got %d", ce.EstimatedBytes())
	}

	if ce.EstimatedCostUSD() != 0.0 {
		t.Fatalf("expected $0 for cache hit, got $%.4f", ce.EstimatedCostUSD())
	}
}

func TestCostEstimate_CalculatesCost(t *testing.T) {
	// 1 TB = 1e12 bytes, $5.00 per TB
	ce := &CostEstimate{
		TotalBytesProcessed: 1_000_000_000_000, // 1 TB
		TotalBytesBilled:    1_000_000_000_000,
		CacheHit:            false,
		StatementType:       "SELECT",
	}

	if ce.EstimatedBytes() != 1_000_000_000_000 {
		t.Fatalf("expected 1TB bytes, got %d", ce.EstimatedBytes())
	}

	cost := ce.EstimatedCostUSD()
	if cost != 5.0 {
		t.Fatalf("expected $5.00 for 1TB, got $%.4f", cost)
	}
}

func TestCostEstimate_StringCacheHit(t *testing.T) {
	ce := &CostEstimate{CacheHit: true}
	s := ce.String()
	if s != "Cache hit — no cost" {
		t.Fatalf("unexpected string: %s", s)
	}
}

func TestCostEstimate_StringWithCost(t *testing.T) {
	ce := &CostEstimate{
		TotalBytesProcessed: 500_000_000_000, // 500 GB
		TotalBytesBilled:    500_000_000_000,
		CacheHit:            false,
	}
	s := ce.String()
	if s == "" {
		t.Fatal("expected non-empty string")
	}
}
