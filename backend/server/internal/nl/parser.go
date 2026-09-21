// Package nl provides natural-language query parsing for the Agente
// Workspace. It uses regex-based template matching to parse common
// query patterns into structured QueryIntent values, covering ~80%
// of expected cases. Unparseable queries are returned as errors so
// they can fall through to the Gemma LLM backend.
package nl

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// QueryIntent — structured representation of a parsed NL query
// ---------------------------------------------------------------------------

// QueryIntent holds the structured interpretation of a natural-language query.
type QueryIntent struct {
	Action    string            `json:"action"`    // select, top, compare, trend, count
	Metric    string            `json:"metric"`    // fcr, nps, tmo, aht, csat, sla, volume, resolution
	Filters   map[string]string `json:"filters"`   // field → value (e.g. agent_name, pliego, team)
	TimeRange string            `json:"time_range"` // last_month, last_week, last_quarter, last_year, or empty
	GroupBy   string            `json:"group_by"`  // agent, pliego, team, date, month, week, day or empty
	Target    string            `json:"target"`    // specific entity value (agent name, pliego name, etc.)
	Limit     int               `json:"limit"`     // row limit (0 = no explicit limit)
	RawQuery  string            `json:"raw_query"` // the original NL text
}

// ---------------------------------------------------------------------------
// Template patterns (ordered by specificity — most specific first)
// ---------------------------------------------------------------------------

type template struct {
	pattern *regexp.Regexp
	build   func(match []string, raw string) *QueryIntent
}

// templates is the ordered list of matching templates.
// It MUST be defined via init() so the patterns compile once.
var templates []template

func init() {
	templates = []template{
		// Pattern 1: "show me <metric> for <entity> in <pliego/team>"
		//   e.g. "show me NPS for agent María in pliego Atención"
		{
			pattern: regexp.MustCompile(`(?i)^show\s+me\s+(\S+)\s+for\s+agent\s+(.+?)\s+in\s+(pliego|team)\s+(.+)$`),
			build: func(m []string, raw string) *QueryIntent {
				qi := &QueryIntent{
					Action:   "select",
					Metric:   normalizeMetric(m[1]),
					Filters:  map[string]string{"agent_name": strings.TrimSpace(m[2])},
					RawQuery: raw,
				}
				qi.Filters[m[3]+"_name"] = strings.TrimSpace(m[4])
				return qi
			},
		},
		// Pattern 2: "show me <metric> for agent <name>"
		//   e.g. "show me FCR for agent John Doe"
		{
			pattern: regexp.MustCompile(`(?i)^show\s+me\s+(\S+)\s+for\s+agent\s+(.+)$`),
			build: func(m []string, raw string) *QueryIntent {
				return &QueryIntent{
					Action:  "select",
					Metric:  normalizeMetric(m[1]),
					Filters: map[string]string{"agent_name": strings.TrimSpace(m[2])},
					RawQuery: raw,
				}
			},
		},
		// Pattern 3: "<metric> last <time_period>"
		//   e.g. "NPS last month", "FCR last quarter", "TMO last week"
		{
			pattern: regexp.MustCompile(`(?i)^(\S+)\s+last\s+(month|week|quarter|year)$`),
			build: func(m []string, raw string) *QueryIntent {
				return &QueryIntent{
					Action:    "select",
					Metric:    normalizeMetric(m[1]),
					TimeRange: "last_" + strings.ToLower(m[2]),
					RawQuery:  raw,
				}
			},
		},
		// Pattern 4: "show me <metric> last <time_period>"
		//   e.g. "show me FCR last month"
		{
			pattern: regexp.MustCompile(`(?i)^show\s+me\s+(\S+)\s+last\s+(month|week|quarter|year)$`),
			build: func(m []string, raw string) *QueryIntent {
				return &QueryIntent{
					Action:    "select",
					Metric:    normalizeMetric(m[1]),
					TimeRange: "last_" + strings.ToLower(m[2]),
					RawQuery:  raw,
				}
			},
		},
		// Pattern 5: "<metric> by <group>"
		//   e.g. "TMO by pliego", "NPS by agent", "CSAT by team"
		{
			pattern: regexp.MustCompile(`(?i)^(\S+)\s+by\s+(pliego|agent|team|date|month|week|day)$`),
			build: func(m []string, raw string) *QueryIntent {
				return &QueryIntent{
					Action:   "select",
					Metric:   normalizeMetric(m[1]),
					GroupBy:  strings.ToLower(m[2]),
					RawQuery: raw,
				}
			},
		},
		// Pattern 6: "show me <metric> by <group>"
		//   e.g. "show me TMO by pliego"
		{
			pattern: regexp.MustCompile(`(?i)^show\s+me\s+(\S+)\s+by\s+(pliego|agent|team|date|month|week|day)$`),
			build: func(m []string, raw string) *QueryIntent {
				return &QueryIntent{
					Action:   "select",
					Metric:   normalizeMetric(m[1]),
					GroupBy:  strings.ToLower(m[2]),
					RawQuery: raw,
				}
			},
		},
		// Pattern 7: "top <N> agents by <metric>"
		//   e.g. "top 10 agents by FCR", "top agents by NPS"
		{
			pattern: regexp.MustCompile(`(?i)^top\s+(\d+\s+)?agents?\s+by\s+(\S+)$`),
			build: func(m []string, raw string) *QueryIntent {
				limit := 10 // default
				if m[1] != "" {
					if n, err := strconv.Atoi(strings.TrimSpace(m[1])); err == nil {
						limit = n
					}
				}
				return &QueryIntent{
					Action:   "top",
					Metric:   normalizeMetric(m[2]),
					GroupBy:  "agent",
					Limit:    limit,
					RawQuery: raw,
				}
			},
		},
		// Pattern 8: "bottom <N> agents by <metric>"
		//   e.g. "bottom 5 agents by TMO"
		{
			pattern: regexp.MustCompile(`(?i)^bottom\s+(\d+\s+)?agents?\s+by\s+(\S+)$`),
			build: func(m []string, raw string) *QueryIntent {
				limit := 10
				if m[1] != "" {
					if n, err := strconv.Atoi(strings.TrimSpace(m[1])); err == nil {
						limit = n
					}
				}
				return &QueryIntent{
					Action:   "bottom",
					Metric:   normalizeMetric(m[2]),
					GroupBy:  "agent",
					Limit:    limit,
					RawQuery: raw,
				}
			},
		},
		// Pattern 9: "<metric> for <pliego>"
		//   e.g. "FCR for pliego Atención"
		{
			pattern: regexp.MustCompile(`(?i)^(\S+)\s+for\s+pliego\s+(.+)$`),
			build: func(m []string, raw string) *QueryIntent {
				return &QueryIntent{
					Action:   "select",
					Metric:   normalizeMetric(m[1]),
					Filters:  map[string]string{"pliego_name": strings.TrimSpace(m[2])},
					RawQuery: raw,
				}
			},
		},
		// Pattern 10: "<metric> for agent <name> this <time_period>"
		//   e.g. "FCR for agent John this month"
		{
			pattern: regexp.MustCompile(`(?i)^(\S+)\s+for\s+agent\s+(.+?)\s+this\s+(month|week|quarter|year)$`),
			build: func(m []string, raw string) *QueryIntent {
				return &QueryIntent{
					Action:    "select",
					Metric:    normalizeMetric(m[1]),
					Filters:   map[string]string{"agent_name": strings.TrimSpace(m[2])},
					TimeRange: "this_" + strings.ToLower(m[3]),
					RawQuery:  raw,
				}
			},
		},
		// Pattern 11: "show me <metric> between <date1> and <date2>"
		//   e.g. "show me FCR between 2024-01-01 and 2024-03-31"
		{
			pattern: regexp.MustCompile(`(?i)^show\s+me\s+(\S+)\s+between\s+(.+?)\s+and\s+(.+)$`),
			build: func(m []string, raw string) *QueryIntent {
				qi := &QueryIntent{
					Action:    "select",
					Metric:    normalizeMetric(m[1]),
					TimeRange: "custom",
					RawQuery:  raw,
				}
				if qi.Filters == nil {
					qi.Filters = make(map[string]string)
				}
				qi.Filters["start_date"] = strings.TrimSpace(m[2])
				qi.Filters["end_date"] = strings.TrimSpace(m[3])
				return qi
			},
		},
		// Pattern 12: "compare <metric> between <entity1> and <entity2>"
		//   e.g. "compare FCR between agent John and agent Maria"
		{
			pattern: regexp.MustCompile(`(?i)^compare\s+(\S+)\s+between\s+(.+?)\s+and\s+(.+)$`),
			build: func(m []string, raw string) *QueryIntent {
				return &QueryIntent{
					Action:   "compare",
					Metric:   normalizeMetric(m[1]),
					Target:   strings.TrimSpace(m[2]) + " vs " + strings.TrimSpace(m[3]),
					RawQuery: raw,
				}
			},
		},
		// Pattern 13: "count <entity> by <group>"
		//   e.g. "count calls by agent", "count interactions by pliego"
		{
			pattern: regexp.MustCompile(`(?i)^count\s+(\S+)\s+by\s+(pliego|agent|team|date|month|week|day)$`),
			build: func(m []string, raw string) *QueryIntent {
				return &QueryIntent{
					Action:   "count",
					Metric:   strings.ToLower(m[1]),
					GroupBy:  strings.ToLower(m[2]),
					RawQuery: raw,
				}
			},
		},
		// Pattern 14: "trend of <metric> over <time_period>"
		//   e.g. "trend of NPS over last 6 months"
		{
			pattern: regexp.MustCompile(`(?i)^trend\s+of\s+(\S+)\s+over\s+(.+)$`),
			build: func(m []string, raw string) *QueryIntent {
				qi := &QueryIntent{
					Action:   "trend",
					Metric:   normalizeMetric(m[1]),
					GroupBy:  "date",
					RawQuery: raw,
				}
				timeDesc := strings.TrimSpace(m[2])
				if strings.Contains(timeDesc, "month") {
					qi.TimeRange = timeDesc
				} else if strings.Contains(timeDesc, "week") {
					qi.TimeRange = timeDesc
				} else if strings.Contains(timeDesc, "quarter") {
					qi.TimeRange = timeDesc
				} else if strings.Contains(timeDesc, "year") {
					qi.TimeRange = timeDesc
				}
				return qi
			},
		},
		// Pattern 15: "<metric> for <time_period>"
		//   e.g. "NPS for last month", "FCR this quarter"
		{
			pattern: regexp.MustCompile(`(?i)^(\S+)\s+for\s+(last|this)\s+(month|week|quarter|year)$`),
			build: func(m []string, raw string) *QueryIntent {
				prefix := "last_"
				if strings.EqualFold(m[2], "this") {
					prefix = "this_"
				}
				return &QueryIntent{
					Action:    "select",
					Metric:    normalizeMetric(m[1]),
					TimeRange: prefix + strings.ToLower(m[3]),
					RawQuery:  raw,
				}
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Metric normalisation — map common aliases to canonical metric names
// ---------------------------------------------------------------------------

var metricAliases = map[string]string{
	"fcr":         "fcr",
	"firstcall":   "fcr",
	"first_call":  "fcr",
	"nps":         "nps",
	"tmo":         "tmo",
	"aht":         "aht",
	"avgholdtime": "aht",
	"csat":        "csat",
	"resolution":  "resolution",
	"sla":         "sla",
	"volume":      "volume",
	"calls":       "volume",
	"interactions":"volume",
	"ocupacion":   "occupancy",
	"occupancy":   "occupancy",
}

func normalizeMetric(m string) string {
	lower := strings.ToLower(strings.TrimSpace(m))
	if canonical, ok := metricAliases[lower]; ok {
		return canonical
	}
	return lower
}

// ---------------------------------------------------------------------------
// Parse — attempt to parse a natural-language query into a QueryIntent
// ---------------------------------------------------------------------------

// ErrUnparseable is returned when none of the known templates match.
type ErrUnparseable struct {
	Raw string
}

func (e *ErrUnparseable) Error() string {
	return fmt.Sprintf("nl: unparseable query %q", e.Raw)
}

// Parse attempts to match the raw natural-language query against the
// template list. It returns the first matching QueryIntent, or an
// ErrUnparseable error if no template matched (for fallback to Gemma).
func Parse(raw string) (*QueryIntent, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, &ErrUnparseable{Raw: raw}
	}

	for _, t := range templates {
		m := t.pattern.FindStringSubmatch(raw)
		if m != nil {
			qi := t.build(m, raw)
			return qi, nil
		}
	}

	return nil, &ErrUnparseable{Raw: raw}
}

// IsUnparseable returns true if err is an ErrUnparseable.
func IsUnparseable(err error) bool {
	_, ok := err.(*ErrUnparseable)
	return ok
}

// MetricTables returns a suggested table name for a given metric,
// based on the available SILVER/GOLD schema knowledge. This provides
// a hint for SQL generation.
func MetricTables(metric string) []string {
	switch metric {
	case "fcr", "firstcall", "first_call":
		return []string{
			"`dev1pruebas.ANALYTICS_GOLD.vw_fcr_agente`",
			"`dev1pruebas.ANALYTICS_GOLD.vw_fcr_pliego`",
			"`dev1pruebas.ANALYTICS_SILVER.interacciones`",
		}
	case "nps":
		return []string{
			"`dev1pruebas.ANALYTICS_GOLD.vw_nps_agente`",
			"`dev1pruebas.ANALYTICS_GOLD.vw_nps_pliego`",
		}
	case "tmo":
		return []string{
			"`dev1pruebas.ANALYTICS_GOLD.vw_tmo_agente`",
			"`dev1pruebas.ANALYTICS_GOLD.vw_tmo_pliego`",
		}
	case "aht":
		return []string{
			"`dev1pruebas.ANALYTICS_GOLD.vw_aht_agente`",
			"`dev1pruebas.ANALYTICS_GOLD.vw_aht_pliego`",
		}
	case "csat":
		return []string{
			"`dev1pruebas.ANALYTICS_GOLD.vw_csat_agente`",
			"`dev1pruebas.ANALYTICS_GOLD.vw_csat_pliego`",
		}
	case "sla":
		return []string{
			"`dev1pruebas.ANALYTICS_GOLD.vw_sla_agente`",
			"`dev1pruebas.ANALYTICS_GOLD.vw_sla_pliego`",
		}
	case "volume", "calls", "interactions":
		return []string{
			"`dev1pruebas.ANALYTICS_SILVER.interacciones`",
		}
	case "resolution":
		return []string{
			"`dev1pruebas.ANALYTICS_GOLD.vw_resolution_agente`",
		}
	default:
		return []string{
			"`dev1pruebas.ANALYTICS_SILVER.interacciones`",
			"`dev1pruebas.ANALYTICS_GOLD.vw_fcr_agente`",
		}
	}
}

// SuggestedSQL builds a best-effort SQL template from the query intent.
// This is used by the fast path; the schema and metric are used to
// build a reasonable query.
func SuggestedSQL(qi *QueryIntent) string {
	tables := MetricTables(qi.Metric)
	if len(tables) == 0 {
		return ""
	}
	table := tables[0] // pick the most specific table

	// Build SELECT clause
	var selectCol string
	switch qi.Metric {
	case "fcr":
		selectCol = "ROUND(AVG(fcr_rate) * 100, 2) AS fcr_pct"
	case "nps":
		selectCol = "ROUND(AVG(nps_score), 2) AS nps"
	case "tmo":
		selectCol = "ROUND(AVG(tmo_minutes), 2) AS tmo_minutes"
	case "aht":
		selectCol = "ROUND(AVG(aht_minutes), 2) AS aht_minutes"
	case "csat":
		selectCol = "ROUND(AVG(csat_score), 2) AS csat"
	case "sla":
		selectCol = "ROUND(AVG(sla_pct), 2) AS sla_pct"
	case "volume", "calls", "interactions":
		selectCol = "COUNT(*) AS total_interactions"
	case "resolution":
		selectCol = "ROUND(AVG(resolution_rate) * 100, 2) AS resolution_pct"
	case "occupancy":
		selectCol = "ROUND(AVG(occupancy_pct), 2) AS occupancy_pct"
	default:
		selectCol = "COUNT(*) AS count"
	}

	if qi.Action == "top" || qi.Action == "bottom" {
		return buildTopBottomSQL(qi, table, selectCol)
	}

	if qi.Action == "count" {
		return buildCountSQL(qi, table)
	}

	if qi.Action == "trend" {
		return buildTrendSQL(qi, table, selectCol)
	}

	// Standard SELECT
	var whereClauses []string
	var groupCol string

	if qi.Filters != nil {
		for field, val := range qi.Filters {
			switch field {
			case "agent_name":
				whereClauses = append(whereClauses, fmt.Sprintf("LOWER(agent_name) LIKE '%%%s%%'", escapeSQLLiteral(strings.ToLower(val))))
			case "pliego_name":
				whereClauses = append(whereClauses, fmt.Sprintf("LOWER(pliego_name) LIKE '%%%s%%'", escapeSQLLiteral(strings.ToLower(val))))
			case "team_name":
				whereClauses = append(whereClauses, fmt.Sprintf("LOWER(team_name) LIKE '%%%s%%'", escapeSQLLiteral(strings.ToLower(val))))
			}
		}
	}

	switch qi.TimeRange {
	case "last_month":
		whereClauses = append(whereClauses, "date_trunc(fecha, month) = date_trunc(date_sub(current_date(), interval 1 month), month)")
	case "last_week":
		whereClauses = append(whereClauses, "date_trunc(fecha, week) = date_trunc(date_sub(current_date(), interval 1 week), week)")
	case "last_quarter":
		whereClauses = append(whereClauses, "date_trunc(fecha, quarter) = date_trunc(date_sub(current_date(), interval 1 quarter), quarter)")
	case "last_year":
		whereClauses = append(whereClauses, "date_trunc(fecha, year) = date_trunc(date_sub(current_date(), interval 1 year), year)")
	case "this_month":
		whereClauses = append(whereClauses, "date_trunc(fecha, month) = date_trunc(current_date(), month)")
	case "this_week":
		whereClauses = append(whereClauses, "date_trunc(fecha, week) = date_trunc(current_date(), week)")
	case "this_quarter":
		whereClauses = append(whereClauses, "date_trunc(fecha, quarter) = date_trunc(current_date(), quarter)")
	case "this_year":
		whereClauses = append(whereClauses, "date_trunc(fecha, year) = date_trunc(current_date(), year)")
	case "custom":
		if qi.Filters != nil {
			if sd, ok := qi.Filters["start_date"]; ok {
				whereClauses = append(whereClauses, fmt.Sprintf("fecha >= '%s'", escapeSQLLiteral(sd)))
			}
			if ed, ok := qi.Filters["end_date"]; ok {
				whereClauses = append(whereClauses, fmt.Sprintf("fecha <= '%s'", escapeSQLLiteral(ed)))
			}
		}
	}

	switch qi.GroupBy {
	case "agent":
		groupCol = "agent_name"
	case "pliego":
		groupCol = "pliego_name"
	case "team":
		groupCol = "team_name"
	case "month":
		groupCol = "date_trunc(fecha, month) AS month"
	case "week":
		groupCol = "date_trunc(fecha, week) AS week"
	case "day", "date":
		groupCol = "fecha AS day"
	}

	var sql strings.Builder
	sql.WriteString("SELECT ")
	if groupCol != "" {
		sql.WriteString(groupCol)
		sql.WriteString(", ")
	}
	sql.WriteString(selectCol)
	sql.WriteString("\nFROM ")
	sql.WriteString(table)

	if len(whereClauses) > 0 {
		sql.WriteString("\nWHERE ")
		sql.WriteString(strings.Join(whereClauses, "\n  AND "))
	}

	if groupCol != "" {
		groupByCol := groupCol
		if idx := strings.Index(groupCol, " AS "); idx > 0 {
			groupByCol = groupCol[:idx]
		}
		sql.WriteString("\nGROUP BY ")
		sql.WriteString(groupByCol)
	}

	sql.WriteString("\nLIMIT 100")

	return sql.String()
}

func buildTopBottomSQL(qi *QueryIntent, table, selectCol string) string {
	order := "DESC"
	if qi.Action == "bottom" {
		order = "ASC"
	}

	limit := qi.Limit
	if limit <= 0 {
		limit = 10
	}

	var groupCol string
	switch qi.GroupBy {
	case "agent":
		groupCol = "agent_name"
	case "pliego":
		groupCol = "pliego_name"
	case "team":
		groupCol = "team_name"
	}

	return fmt.Sprintf(
		"SELECT %s, %s\nFROM %s\nGROUP BY %s\nORDER BY %s %s\nLIMIT %d",
		groupCol, selectCol, table, groupCol, extractMetricAlias(selectCol), order, limit,
	)
}

func buildCountSQL(qi *QueryIntent, table string) string {
	var groupCol string
	switch qi.GroupBy {
	case "agent":
		groupCol = "agent_name"
	case "pliego":
		groupCol = "pliego_name"
	case "team":
		groupCol = "team_name"
	case "month":
		groupCol = "date_trunc(fecha, month)"
	case "week":
		groupCol = "date_trunc(fecha, week)"
	case "day", "date":
		groupCol = "fecha"
	default:
		groupCol = "agent_name"
	}

	metric := qi.Metric
	if metric == "" {
		metric = "interactions"
	}

	return fmt.Sprintf(
		"SELECT %s, COUNT(*) AS total_%s\nFROM %s\nGROUP BY %s\nORDER BY total_%s DESC\nLIMIT 100",
		groupCol, metric, table, groupCol, metric,
	)
}

func buildTrendSQL(qi *QueryIntent, table, selectCol string) string {
	dateCol := "date_trunc(fecha, month) AS period"
	groupBy := "date_trunc(fecha, month)"
	orderBy := "period"

	return fmt.Sprintf(
		"SELECT %s, %s\nFROM %s\nGROUP BY %s\nORDER BY %s ASC\nLIMIT 100",
		dateCol, selectCol, table, groupBy, orderBy,
	)
}

func extractMetricAlias(selectCol string) string {
	if idx := strings.LastIndex(selectCol, " AS "); idx >= 0 {
		return strings.TrimSpace(selectCol[idx+4:])
	}
	return selectCol
}

func escapeSQLLiteral(s string) string {
	return strings.ReplaceAll(s, "'", "\\'")
}
