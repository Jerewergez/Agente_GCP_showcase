package nl

import (
	"testing"
	"time"
)

// TestSchemaCache uses table-driven tests for cache operations without
// requiring an actual BigQuery connection.
func TestSchemaCacheDefaults(t *testing.T) {
	sc := NewSchemaCache(nil, []string{"test_dataset"})
	if sc == nil {
		t.Fatal("expected non-nil SchemaCache")
	}
	if sc.ttl != 1*time.Hour {
		t.Fatalf("expected default TTL 1h, got %v", sc.ttl)
	}
	if sc.refreshInterval != 1*time.Hour {
		t.Fatalf("expected default refresh interval 1h, got %v", sc.refreshInterval)
	}
}

func TestSchemaCacheWithOptions(t *testing.T) {
	sc := NewSchemaCache(nil, []string{"ds1"},
		WithTTL(30*time.Minute),
		WithRefreshInterval(10*time.Minute),
	)
	if sc.ttl != 30*time.Minute {
		t.Fatalf("expected TTL 30m, got %v", sc.ttl)
	}
	if sc.refreshInterval != 10*time.Minute {
		t.Fatalf("expected refresh interval 10m, got %v", sc.refreshInterval)
	}
}

func TestSchemaCacheEmpty(t *testing.T) {
	sc := NewSchemaCache(nil, nil)
	schemas := sc.GetAllSchemas()
	if schemas == nil {
		t.Fatal("expected non-nil schemas map")
	}
	if len(schemas) != 0 {
		t.Fatalf("expected empty schemas, got %d entries", len(schemas))
	}
}

func TestSchemaCacheIsStale(t *testing.T) {
	sc := NewSchemaCache(nil, []string{"ds"})
	if !sc.IsStale() {
		t.Fatal("expected cache to be stale before first load")
	}
}

func TestSchemaCacheGetSchemaUnknown(t *testing.T) {
	sc := NewSchemaCache(nil, []string{"real_ds"})
	schemas := sc.GetSchema("nonexistent")
	if schemas != nil {
		t.Fatal("expected nil for unknown dataset")
	}
}

func TestSchemaCacheStatsInitial(t *testing.T) {
	sc := NewSchemaCache(nil, []string{"ds1", "ds2"})
	stats := sc.Stats()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats["datasets"].(int) != 0 {
		t.Fatalf("expected 0 datasets, got %d", stats["datasets"])
	}
}

func TestSchemaCacheGetSchemaDescription(t *testing.T) {
	sc := NewSchemaCache(nil, []string{"test_ds"})
	desc := sc.GetSchemaDescription()
	// Should be empty since the cache has not been loaded
	// (no BigQuery connection), but should not panic
	_ = desc
}

func TestSchemaCacheLastLoad(t *testing.T) {
	sc := NewSchemaCache(nil, []string{"ds"})
	load := sc.LastLoad()
	if !load.IsZero() {
		t.Fatal("expected zero time before first load")
	}
}
