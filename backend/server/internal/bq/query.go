package bq

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/bigquery"
)

// QueryBuilder builds safe parameterized BigQuery queries.
type QueryBuilder struct {
	query string
	args  []interface{}
}

// NewQuery creates a new QueryBuilder with the given SQL template.
// Use ? placeholders for parameters, then call WithArg to bind values.
func NewQuery(sql string) *QueryBuilder {
	return &QueryBuilder{query: sql}
}

// WithArg binds a parameter value to the query. The order of calls
// must match the ? placeholders in the SQL template.
func (qb *QueryBuilder) WithArg(v interface{}) *QueryBuilder {
	qb.args = append(qb.args, v)
	return qb
}

// WithArgs binds multiple parameter values at once.
func (qb *QueryBuilder) WithArgs(vals ...interface{}) *QueryBuilder {
	qb.args = append(qb.args, vals...)
	return qb
}

// SQL returns the parameterised SQL string and arguments.
func (qb *QueryBuilder) SQL() (string, []interface{}) {
	return qb.query, qb.args
}

// Run executes the query via the BigQuery client and returns an
// iterator over the result rows.
func (c *Client) Run(ctx context.Context, qb *QueryBuilder) (*bigquery.RowIterator, error) {
	sql, args := qb.SQL()
	q := c.client.Query(sql)
	q.Parameters = make([]bigquery.QueryParameter, len(args))
	for i, arg := range args {
		q.Parameters[i] = bigquery.QueryParameter{Value: arg}
	}

	job, err := q.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("bq query run: %w", err)
	}

	log.Printf("bq: job %s submitted for project %s", job.ID(), c.ProjectID)
	it, err := job.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("bq job read: %w", err)
	}
	return it, nil
}

// RunDry executes the query in dry-run mode to validate syntax and
// estimate cost without consuming any slot time.
func (c *Client) RunDry(ctx context.Context, qb *QueryBuilder) (*CostEstimate, error) {
	sql, args := qb.SQL()
	q := c.client.Query(sql)
	q.Parameters = make([]bigquery.QueryParameter, len(args))
	for i, arg := range args {
		q.Parameters[i] = bigquery.QueryParameter{Value: arg}
	}
	q.DryRun = true

	job, err := q.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("bq dry run: %w", err)
	}

	status := job.LastStatus()
	if status == nil || status.Statistics == nil || status.Statistics.Details == nil {
		return nil, fmt.Errorf("bq dry run: no statistics returned")
	}

	queryStats, ok := status.Statistics.Details.(*bigquery.QueryStatistics)
	if !ok {
		return nil, fmt.Errorf("bq dry run: unexpected statistics type")
	}

	estimate := &CostEstimate{
		TotalBytesProcessed: queryStats.TotalBytesProcessed,
		TotalBytesBilled:    queryStats.TotalBytesBilled,
		CacheHit:            queryStats.CacheHit,
		StatementType:       queryStats.StatementType,
	}

	log.Printf("bq: dry-run estimate — bytes processed=%d bytes billed=%d cache=%v stmt=%s",
		estimate.TotalBytesProcessed, estimate.TotalBytesBilled, estimate.CacheHit, estimate.StatementType)

	return estimate, nil
}
