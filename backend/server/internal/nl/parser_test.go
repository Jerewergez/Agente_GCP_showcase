package nl

import (
	"testing"
)

func TestParse_ShowMeMetricForAgent(t *testing.T) {
	qi, err := Parse("show me FCR for agent John Doe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Action != "select" {
		t.Fatalf("expected action 'select', got %q", qi.Action)
	}
	if qi.Metric != "fcr" {
		t.Fatalf("expected metric 'fcr', got %q", qi.Metric)
	}
	if qi.Filters["agent_name"] != "John Doe" {
		t.Fatalf("expected agent_name 'John Doe', got %q", qi.Filters["agent_name"])
	}
}

func TestParse_MetricLastMonth(t *testing.T) {
	qi, err := Parse("NPS last month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Metric != "nps" {
		t.Fatalf("expected metric 'nps', got %q", qi.Metric)
	}
	if qi.TimeRange != "last_month" {
		t.Fatalf("expected time_range 'last_month', got %q", qi.TimeRange)
	}
}

func TestParse_MetricLastWeek(t *testing.T) {
	qi, err := Parse("TMO last week")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Metric != "tmo" {
		t.Fatalf("expected metric 'tmo', got %q", qi.Metric)
	}
	if qi.TimeRange != "last_week" {
		t.Fatalf("expected time_range 'last_week', got %q", qi.TimeRange)
	}
}

func TestParse_MetricByPliego(t *testing.T) {
	qi, err := Parse("TMO by pliego")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Metric != "tmo" {
		t.Fatalf("expected metric 'tmo', got %q", qi.Metric)
	}
	if qi.GroupBy != "pliego" {
		t.Fatalf("expected group_by 'pliego', got %q", qi.GroupBy)
	}
}

func TestParse_TopNAgentsByMetric(t *testing.T) {
	qi, err := Parse("top 5 agents by FCR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Action != "top" {
		t.Fatalf("expected action 'top', got %q", qi.Action)
	}
	if qi.Metric != "fcr" {
		t.Fatalf("expected metric 'fcr', got %q", qi.Metric)
	}
	if qi.Limit != 5 {
		t.Fatalf("expected limit 5, got %d", qi.Limit)
	}
}

func TestParse_TopAgentsDefaultLimit(t *testing.T) {
	qi, err := Parse("top agents by NPS")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Action != "top" {
		t.Fatalf("expected action 'top', got %q", qi.Action)
	}
	if qi.Metric != "nps" {
		t.Fatalf("expected metric 'nps', got %q", qi.Metric)
	}
	if qi.Limit != 10 {
		t.Fatalf("expected default limit 10, got %d", qi.Limit)
	}
}

func TestParse_BottomAgents(t *testing.T) {
	qi, err := Parse("bottom 3 agents by TMO")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Action != "bottom" {
		t.Fatalf("expected action 'bottom', got %q", qi.Action)
	}
	if qi.Limit != 3 {
		t.Fatalf("expected limit 3, got %d", qi.Limit)
	}
}

func TestParse_ShowMeMetricForPliego(t *testing.T) {
	qi, err := Parse("show me NPS for agent Maria in pliego Atencion")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Metric != "nps" {
		t.Fatalf("expected metric 'nps', got %q", qi.Metric)
	}
	if qi.Filters["agent_name"] != "Maria" {
		t.Fatalf("expected agent_name 'Maria', got %q", qi.Filters["agent_name"])
	}
	if qi.Filters["pliego_name"] != "Atencion" {
		t.Fatalf("expected pliego_name 'Atencion', got %q", qi.Filters["pliego_name"])
	}
}

func TestParse_MetricForPliego(t *testing.T) {
	qi, err := Parse("FCR for pliego Soporte")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Metric != "fcr" {
		t.Fatalf("expected metric 'fcr', got %q", qi.Metric)
	}
	if qi.Filters["pliego_name"] != "Soporte" {
		t.Fatalf("expected pliego_name 'Soporte', got %q", qi.Filters["pliego_name"])
	}
}

func TestParse_ShowMeMetricBetweenDates(t *testing.T) {
	qi, err := Parse("show me FCR between 2024-01-01 and 2024-03-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.TimeRange != "custom" {
		t.Fatalf("expected time_range 'custom', got %q", qi.TimeRange)
	}
	if qi.Filters["start_date"] != "2024-01-01" {
		t.Fatalf("expected start_date '2024-01-01', got %q", qi.Filters["start_date"])
	}
	if qi.Filters["end_date"] != "2024-03-31" {
		t.Fatalf("expected end_date '2024-03-31', got %q", qi.Filters["end_date"])
	}
}

func TestParse_CompareMetric(t *testing.T) {
	qi, err := Parse("compare FCR between agent John and agent Maria")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Action != "compare" {
		t.Fatalf("expected action 'compare', got %q", qi.Action)
	}
	if qi.Metric != "fcr" {
		t.Fatalf("expected metric 'fcr', got %q", qi.Metric)
	}
	if qi.Target != "agent John vs agent Maria" {
		t.Fatalf("expected target 'agent John vs agent Maria', got %q", qi.Target)
	}
}

func TestParse_CountByGroup(t *testing.T) {
	qi, err := Parse("count calls by agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Action != "count" {
		t.Fatalf("expected action 'count', got %q", qi.Action)
	}
	if qi.GroupBy != "agent" {
		t.Fatalf("expected group_by 'agent', got %q", qi.GroupBy)
	}
}

func TestParse_TrendOfMetric(t *testing.T) {
	qi, err := Parse("trend of NPS over last 6 months")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Action != "trend" {
		t.Fatalf("expected action 'trend', got %q", qi.Action)
	}
	if qi.Metric != "nps" {
		t.Fatalf("expected metric 'nps', got %q", qi.Metric)
	}
	if qi.GroupBy != "date" {
		t.Fatalf("expected group_by 'date', got %q", qi.GroupBy)
	}
}

func TestParse_MetricForTimePeriod(t *testing.T) {
	qi, err := Parse("NPS for last quarter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Metric != "nps" {
		t.Fatalf("expected metric 'nps', got %q", qi.Metric)
	}
	if qi.TimeRange != "last_quarter" {
		t.Fatalf("expected time_range 'last_quarter', got %q", qi.TimeRange)
	}
}

func TestParse_ShowMeMetricByGroup(t *testing.T) {
	qi, err := Parse("show me CSAT by team")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Metric != "csat" {
		t.Fatalf("expected metric 'csat', got %q", qi.Metric)
	}
	if qi.GroupBy != "team" {
		t.Fatalf("expected group_by 'team', got %q", qi.GroupBy)
	}
}

func TestParse_MetricForAgentThisPeriod(t *testing.T) {
	qi, err := Parse("FCR for agent John this month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Metric != "fcr" {
		t.Fatalf("expected metric 'fcr', got %q", qi.Metric)
	}
	if qi.Filters["agent_name"] != "John" {
		t.Fatalf("expected agent_name 'John', got %q", qi.Filters["agent_name"])
	}
	if qi.TimeRange != "this_month" {
		t.Fatalf("expected time_range 'this_month', got %q", qi.TimeRange)
	}
}

func TestParse_EmptyString(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
	if !IsUnparseable(err) {
		t.Fatalf("expected ErrUnparseable, got %T", err)
	}
}

func TestParse_UnknownQuery(t *testing.T) {
	_, err := Parse("what is the meaning of life?")
	if err == nil {
		t.Fatal("expected error for unknown query")
	}
	if !IsUnparseable(err) {
		t.Fatalf("expected ErrUnparseable, got %T", err)
	}
}

func TestParse_CaseInsensitive(t *testing.T) {
	qi, err := Parse("SHOW ME FCR FOR AGENT MARIA")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Metric != "fcr" {
		t.Fatalf("expected metric 'fcr', got %q", qi.Metric)
	}
	if qi.Filters["agent_name"] != "MARIA" {
		t.Fatalf("expected agent_name 'MARIA', got %q", qi.Filters["agent_name"])
	}
}

func TestParse_ShowMeMetricLastMonth(t *testing.T) {
	qi, err := Parse("show me FCR last month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qi.Metric != "fcr" {
		t.Fatalf("expected metric 'fcr', got %q", qi.Metric)
	}
	if qi.TimeRange != "last_month" {
		t.Fatalf("expected time_range 'last_month', got %q", qi.TimeRange)
	}
}

func TestNormalizeMetric(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"FCR", "fcr"},
		{"firstcall", "fcr"},
		{"first_call", "fcr"},
		{"nps", "nps"},
		{"NPS", "nps"},
		{"tmo", "tmo"},
		{"aht", "aht"},
		{"avgholdtime", "aht"},
		{"csat", "csat"},
		{"sla", "sla"},
		{"volume", "volume"},
		{"calls", "volume"},
		{"occupancy", "occupancy"},
		{"unknownmetric", "unknownmetric"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeMetric(tt.input)
			if got != tt.expected {
				t.Fatalf("normalizeMetric(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMetricTables(t *testing.T) {
	tests := []struct {
		metric      string
		expectCount int
	}{
		{"fcr", 3},
		{"nps", 2},
		{"tmo", 2},
		{"aht", 2},
		{"volume", 1},
		{"unknown", 2},
	}
	for _, tt := range tests {
		t.Run(tt.metric, func(t *testing.T) {
			tables := MetricTables(tt.metric)
			if len(tables) < 1 {
				t.Fatalf("expected at least 1 table, got %d", len(tables))
			}
		})
		_ = tt.expectCount
	}
}

func TestSuggestedSQL_Select(t *testing.T) {
	qi, err := Parse("show me FCR for agent Juan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sql := SuggestedSQL(qi)
	if sql == "" {
		t.Fatal("expected non-empty SQL")
	}
	if !containsIgnoreCase(sql, "fcr") {
		t.Fatalf("expected SQL to contain 'fcr', got: %s", sql)
	}
	if !containsIgnoreCase(sql, "juan") {
		t.Fatalf("expected SQL to contain 'Juan', got: %s", sql)
	}
}

func TestSuggestedSQL_Top(t *testing.T) {
	qi, err := Parse("top 5 agents by NPS")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sql := SuggestedSQL(qi)
	if sql == "" {
		t.Fatal("expected non-empty SQL")
	}
	if !containsIgnoreCase(sql, "LIMIT 5") {
		t.Fatalf("expected LIMIT 5, got: %s", sql)
	}
}

func TestEscapeSQLLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"it's", "it\\'s"},
		{"no'escape", "no\\'escape"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeSQLLiteral(tt.input)
			if got != tt.expected {
				t.Fatalf("escapeSQLLiteral(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func containsIgnoreCase(s, substr string) bool {
	s, substr = toUpper(s), toUpper(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toUpper(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		b[i] = c
	}
	return string(b)
}
