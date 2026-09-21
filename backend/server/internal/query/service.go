// Package query orchestrates the NL→SQL pipeline: fast path via
// template matching, slow path via Gemma LLM, then executes against
// BigQuery and returns structured results.
package query

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/Jerewergez/Agente_Workspace/internal/bq"
	"github.com/Jerewergez/Agente_Workspace/internal/ml"
	"github.com/Jerewergez/Agente_Workspace/internal/nl"
)

// ---------------------------------------------------------------------------
// QueryResult — structured result returned by the service
// ---------------------------------------------------------------------------

// QueryResult holds the full output of an NL→SQL query cycle.
type QueryResult struct {
	SQL       string            `json:"sql"`                 // the generated SQL
	Rows      []json.RawMessage `json:"rows"`                // result rows as JSON
	Schema    []ColumnMeta      `json:"schema"`              // column metadata
	LatencyMs int64             `json:"latency_ms"`          // total latency in ms
	Source    string            `json:"source"`              // "template" or "gemma"
	RowCount  int               `json:"row_count"`           // number of rows returned
	Intent    *nl.QueryIntent   `json:"intent,omitempty"`    // parsed intent (fast path)
	Error     string            `json:"error,omitempty"`     // error message if any
}

// ColumnMeta describes a result column.
type ColumnMeta struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ---------------------------------------------------------------------------
// Service — the NL→SQL orchestrator
// ---------------------------------------------------------------------------

// ServiceConfig holds optional configuration for the query service.
type ServiceConfig struct {
	// MaxRows limits the number of result rows returned (default 1000).
	MaxRows int
	// DefaultDataset is used when no dataset context is available.
	DefaultDataset string
}

// Service orchestrates the NL→SQL pipeline.
type Service struct {
	bqClient  *bq.Client
	schema    *nl.SchemaCache
	gemma     *ml.GemmaClient
	cfg       ServiceConfig
}

// NewService creates an NL→SQL service with the given dependencies.
func NewService(bqClient *bq.Client, schema *nl.SchemaCache, gemma *ml.GemmaClient, cfg ServiceConfig) *Service {
	if cfg.MaxRows <= 0 {
		cfg.MaxRows = 1000
	}
	return &Service{
		bqClient: bqClient,
		schema:   schema,
		gemma:    gemma,
		cfg:      cfg,
	}
}

// Query processes a natural-language query and returns structured results.
//
// Fast path: Try the regex template matcher. On success, generate SQL
// from the parsed intent and execute it against BigQuery.
//
// Slow path: If the template matcher fails AND Gemma is configured,
// build a prompt with schema context, call Gemma for SQL generation,
// then execute the returned SQL.
func (s *Service) Query(ctx context.Context, rawQuery string) (*QueryResult, error) {
	start := time.Now()

	// ---- Fast pass: template matcher ----
	intent, err := nl.Parse(rawQuery)
	var sqlStr string

	if err == nil {
		// Template matched — generate SQL from intent
		sqlStr = nl.SuggestedSQL(intent)
		if sqlStr == "" {
			return nil, fmt.Errorf("query: template matched but could not generate SQL for intent %+v", intent)
		}
		log.Printf("query: fast path — template matched metric=%s action=%s", intent.Metric, intent.Action)
	} else if nl.IsUnparseable(err) && s.gemma != nil && s.gemma.IsConfigured() {
		// ---- Slow path: Gemma fallback ----
		prompt := s.buildGemmaPrompt(rawQuery)
		gemmaResp, gemmaErr := s.gemma.Predict(ctx, prompt,
			ml.WithMaxTokens(1024),
			ml.WithTemperature(0.2),
		)
		if gemmaErr != nil {
			return nil, fmt.Errorf("query: gemma fallback failed: %w", gemmaErr)
		}

		// Extract SQL from Gemma response (it may be wrapped in markdown)
		sqlStr = extractSQLFromGemma(gemmaResp.Text)
		if sqlStr == "" {
			return nil, fmt.Errorf("query: gemma returned empty SQL: %q", gemmaResp.Text)
		}
		log.Printf("query: slow path — gemma generated SQL (confidence=%.2f)", gemmaResp.Confidence)
	} else {
		return nil, fmt.Errorf("query: cannot process — template failed (%v) and gemma unavailable", err)
	}

	// ---- Execute the SQL ----
	rows, colMeta, err := s.executeSQL(ctx, sqlStr)
	latencyMs := time.Since(start).Milliseconds()

	// Determine source
	source := "gemma"
	if intent != nil {
		source = "template"
	}

	return &QueryResult{
		SQL:       sqlStr,
		Rows:      rows,
		Schema:    colMeta,
		LatencyMs: latencyMs,
		RowCount:  len(rows),
		Source:    source,
		Intent:    intent,
		Error:     errToString(err),
	}, nil
}

// errToString converts an error to a string, returning empty string for nil.
func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// sourceLabel returns the source label.
func sourceLabel(templateSuccess bool) string {
	if templateSuccess {
		return "template"
	}
	return "gemma"
}

// executeSQL runs the SQL string against BigQuery and returns JSON rows
// plus column metadata.
func (s *Service) executeSQL(ctx context.Context, sqlStr string) ([]json.RawMessage, []ColumnMeta, error) {
	qb := bq.NewQuery(sqlStr)
	it, err := s.bqClient.Run(ctx, qb)
	if err != nil {
		return nil, nil, fmt.Errorf("execute query: %w", err)
	}

	// Read schema from the iterator
	var colMeta []ColumnMeta
	for _, f := range it.Schema {
		colMeta = append(colMeta, ColumnMeta{
			Name: f.Name,
			Type: string(f.Type),
		})
	}

	// Read all rows
	var rows []json.RawMessage
	rowCount := 0
	for {
		var row map[string]bigquery.Value
		err := it.Next(&row)
		if err == sql.ErrNoRows || err == context.Canceled {
			break
		}
		if err != nil {
			// Iterator exhausted
			break
		}

		// Convert bigquery.Value values to plain JSON
		jsonRow := make(map[string]interface{})
		for k, v := range row {
			jsonRow[k] = convertBQValue(v)
		}
		raw, err := json.Marshal(jsonRow)
		if err != nil {
			continue // skip rows that can't be serialised
		}

		rows = append(rows, raw)
		rowCount++

		if rowCount >= s.cfg.MaxRows {
			break
		}
	}

	if rows == nil {
		rows = []json.RawMessage{}
	}
	if colMeta == nil {
		colMeta = []ColumnMeta{}
	}

	return rows, colMeta, nil
}

// convertBQValue converts a bigquery.Value to a plain Go value suitable
// for JSON serialisation.
func convertBQValue(v interface{}) interface{} {
	switch val := v.(type) {
	case time.Time:
		return val.Format(time.RFC3339)
	case []byte:
		return string(val)
	case map[string]bigquery.Value:
		m := make(map[string]interface{})
		for k, vv := range val {
			m[k] = convertBQValue(vv)
		}
		return m
	case []interface{}:
		for i, item := range val {
			val[i] = convertBQValue(item)
		}
		return val
	default:
		return v
	}
}

// buildGemmaPrompt constructs a prompt for the Gemma model that includes
// the schema context and the user's natural-language query.
func (s *Service) buildGemmaPrompt(rawQuery string) string {
	var b strings.Builder

	b.WriteString("You are a BigQuery SQL expert for a contact center analytics system.\n\n")
	b.WriteString("## Schema Context\n")
	b.WriteString(s.schema.GetSchemaDescription())
	b.WriteString("\n")

	b.WriteString("## Rules\n")
	b.WriteString("- Generate only a single SQL query — no explanations, no markdown formatting.\n")
	b.WriteString("- Use standard BigQuery SQL syntax.\n")
	b.WriteString("- The query MUST be a SELECT statement.\n")
	b.WriteString("- Use backtick-quoted table names like `project.dataset.table`.\n")
	b.WriteString("- Limit results to 100 rows unless the user asks for more.\n")
	b.WriteString("- For time-based filters, use the 'fecha' column which holds DATE values.\n")
	b.WriteString("- For agent-level aggregation use agent_name column.\n")
	b.WriteString("- For pliego-level aggregation use pliego_name column.\n")
	b.WriteString("\n## User Query\n")
	b.WriteString(rawQuery)
	b.WriteString("\n\n## SQL\n")

	return b.String()
}

// extractSQLFromGemma attempts to extract a clean SQL statement from the
// Gemma response text, handling markdown code blocks if present.
func extractSQLFromGemma(text string) string {
	text = strings.TrimSpace(text)

	// Try to extract from markdown code block first
	if idx := strings.Index(text, "```sql"); idx >= 0 {
		rest := text[idx+6:]
		if end := strings.Index(rest, "```"); end >= 0 {
			return strings.TrimSpace(rest[:end])
		}
	}
	if idx := strings.Index(text, "```"); idx >= 0 {
		rest := text[idx+3:]
		if end := strings.Index(rest, "```"); end >= 0 {
			return strings.TrimSpace(rest[:end])
		}
		// Maybe only opening fence
		return strings.TrimSpace(rest)
	}

	// No markdown fencing — return the text as-is if it starts with SELECT
	upper := strings.ToUpper(strings.TrimSpace(text))
	if strings.HasPrefix(upper, "SELECT") {
		return text
	}

	// Try to find the first SELECT statement
	if idx := strings.Index(upper, "SELECT"); idx >= 0 {
		return text[idx:]
	}

	return ""
}
