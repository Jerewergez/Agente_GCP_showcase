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
// SchemaCache — in-memory cache over BigQuery INFORMATION_SCHEMA.COLUMNS
// ---------------------------------------------------------------------------

// ColumnInfo holds a single column's metadata.
type ColumnInfo struct {
	TableName  string `json:"table_name"`
	ColumnName string `json:"column_name"`
	DataType   string `json:"data_type"`
	IsNullable string `json:"is_nullable"`
	Comment    string `json:"comment,omitempty"`
}

// TableSchema holds the schema for a single table.
type TableSchema struct {
	TableName string       `json:"table_name"`
	Columns   []ColumnInfo `json:"columns"`
}

// SchemaCache is a thread-safe in-memory cache of BigQuery table schemas.
// It refreshes periodically with a configurable TTL (default 1 hour).
type SchemaCache struct {
	client   *bq.Client
	datasets []string

	mu       sync.RWMutex
	schemas  map[string][]TableSchema // dataset → tables
	lastLoad time.Time
	ttl      time.Duration

	refreshInterval time.Duration
	stopCh          chan struct{}
}

// SchemaCacheOption allows configuring the cache.
type SchemaCacheOption func(*SchemaCache)

// WithTTL sets a custom TTL for cache freshness.
func WithTTL(d time.Duration) SchemaCacheOption {
	return func(sc *SchemaCache) {
		sc.ttl = d
	}
}

// WithRefreshInterval sets a background refresh interval.
func WithRefreshInterval(d time.Duration) SchemaCacheOption {
	return func(sc *SchemaCache) {
		sc.refreshInterval = d
	}
}

// NewSchemaCache creates a schema cache that fetches column metadata
// from the given BigQuery datasets on startup and refreshes periodically.
func NewSchemaCache(client *bq.Client, datasets []string, opts ...SchemaCacheOption) *SchemaCache {
	sc := &SchemaCache{
		client:          client,
		datasets:        datasets,
		schemas:         make(map[string][]TableSchema),
		ttl:             1 * time.Hour,
		refreshInterval: 1 * time.Hour,
		stopCh:          make(chan struct{}),
	}
	for _, o := range opts {
		o(sc)
	}
	return sc
}

// Start initialises the cache by loading data immediately, then starts
// a background refresh goroutine. Call Stop to clean up.
func (sc *SchemaCache) Start(ctx context.Context) error {
	if err := sc.Refresh(ctx); err != nil {
		return fmt.Errorf("schema cache initial load: %w", err)
	}

	go sc.refreshLoop(ctx)
	return nil
}

// Stop halts the background refresh goroutine.
func (sc *SchemaCache) Stop() {
	close(sc.stopCh)
}

// refreshLoop periodically refreshes the cache.
func (sc *SchemaCache) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(sc.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := sc.Refresh(ctx); err != nil {
				log.Printf("nl/schema: background refresh error: %v", err)
			}
		case <-sc.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Refresh forces an immediate reload of schema data from BigQuery.
func (sc *SchemaCache) Refresh(ctx context.Context) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for _, ds := range sc.datasets {
		schemas, err := sc.loadDatasetSchema(ctx, ds)
		if err != nil {
			log.Printf("nl/schema: error loading dataset %q: %v", ds, err)
			continue // keep any cached data for this dataset
		}
		sc.schemas[ds] = schemas
	}

	sc.lastLoad = time.Now()
	log.Printf("nl/schema: refreshed %d datasets", len(sc.datasets))
	return nil
}

// backtickEscape wraps an identifier in backticks for BigQuery SQL.
func backtickEscape(s string) string {
	return "`" + s + "`"
}

// loadDatasetSchema queries INFORMATION_SCHEMA.COLUMNS for a dataset.
func (sc *SchemaCache) loadDatasetSchema(ctx context.Context, datasetID string) ([]TableSchema, error) {
	if strings.Contains(datasetID, ".") {
		// Fully qualified project.dataset
		parts := strings.SplitN(datasetID, ".", 2)
		datasetID = parts[1]
	}

	query := fmt.Sprintf(
		`SELECT table_name, column_name, data_type, is_nullable
		 FROM %s.INFORMATION_SCHEMA.COLUMNS
		 ORDER BY table_name, ordinal_position`,
		backtickEscape(datasetID),
	)

	qb := bq.NewQuery(query)
	it, err := sc.client.Run(ctx, qb)
	if err != nil {
		return nil, fmt.Errorf("query INFORMATION_SCHEMA.COLUMNS for %q: %w", datasetID, err)
	}

	type rawCol struct {
		TableName  string
		ColumnName string
		DataType   string
		IsNullable string
	}

	tableCols := make(map[string][]ColumnInfo)
	for {
		var row rawCol
		err := it.Next(&row)
		if err != nil {
			break
		}
		// Strip the project-qualified prefix for cleaner names
		tableName := row.TableName
		if idx := strings.LastIndex(tableName, "."); idx >= 0 {
			tableName = tableName[idx+1:]
		}

		tableCols[tableName] = append(tableCols[tableName], ColumnInfo{
			TableName:  tableName,
			ColumnName: row.ColumnName,
			DataType:   row.DataType,
			IsNullable: row.IsNullable,
		})
	}

	var result []TableSchema
	for tname, cols := range tableCols {
		result = append(result, TableSchema{
			TableName: tname,
			Columns:   cols,
		})
	}

	if result == nil {
		result = []TableSchema{}
	}
	return result, nil
}

// GetSchema returns the cached schema for a specific dataset.
// Returns nil if the dataset is not cached.
func (sc *SchemaCache) GetSchema(dataset string) []TableSchema {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	if schemas, ok := sc.schemas[dataset]; ok {
		return schemas
	}
	return nil
}

// GetAllSchemas returns schemas for all cached datasets as a flat list.
func (sc *SchemaCache) GetAllSchemas() map[string][]TableSchema {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	result := make(map[string][]TableSchema, len(sc.schemas))
	for k, v := range sc.schemas {
		result[k] = v
	}
	return result
}

// GetSchemaDescription returns a human-readable description of all
// cached schemas, suitable for including in an LLM prompt context.
func (sc *SchemaCache) GetSchemaDescription() string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var b strings.Builder
	for ds, tables := range sc.schemas {
		b.WriteString(fmt.Sprintf("Dataset: %s\n", ds))
		for _, t := range tables {
			b.WriteString(fmt.Sprintf("  Table: %s\n", t.TableName))
			for _, c := range t.Columns {
				nullable := ""
				if c.IsNullable == "YES" {
					nullable = " NULLABLE"
				}
				b.WriteString(fmt.Sprintf("    - %s (%s%s)\n", c.ColumnName, c.DataType, nullable))
			}
		}
	}
	return b.String()
}

// IsStale returns true if the cache has never been loaded or if the
// TTL has expired since the last refresh.
func (sc *SchemaCache) IsStale() bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.lastLoad.IsZero() || time.Since(sc.lastLoad) > sc.ttl
}

// LastLoad returns the time of the last successful refresh.
func (sc *SchemaCache) LastLoad() time.Time {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.lastLoad
}

// Stats returns cache statistics for monitoring.
func (sc *SchemaCache) Stats() map[string]interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	totalTables := 0
	totalColumns := 0
	for _, tables := range sc.schemas {
		for _, t := range tables {
			totalTables++
			totalColumns += len(t.Columns)
		}
	}

	return map[string]interface{}{
		"datasets":    len(sc.schemas),
		"tables":      totalTables,
		"columns":     totalColumns,
		"last_load":   sc.lastLoad.Format(time.RFC3339),
		"ttl_seconds": int(sc.ttl.Seconds()),
	}
}
