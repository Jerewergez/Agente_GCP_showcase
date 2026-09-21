package bq

import "fmt"

// CostEstimate holds the billing information for a BigQuery query.
type CostEstimate struct {
	TotalBytesProcessed int64  `json:"totalBytesProcessed"`
	TotalBytesBilled    int64  `json:"totalBytesBilled"`
	CacheHit            bool   `json:"cacheHit"`
	StatementType       string `json:"statementType"`
}

// EstimatedBytes returns the estimated bytes billed.
// If the query was a cache hit, returns 0.
func (ce *CostEstimate) EstimatedBytes() int64 {
	if ce.CacheHit {
		return 0
	}
	return ce.TotalBytesBilled
}

// EstimatedCostUSD returns the estimated cost in US dollars based on
// the standard BigQuery on-demand pricing ($5 per TB processed).
func (ce *CostEstimate) EstimatedCostUSD() float64 {
	bytes := ce.EstimatedBytes()
	if bytes <= 0 {
		return 0
	}
	// BigQuery on-demand: $5.00 per TB ($5.00 / 1e12 bytes)
	return float64(bytes) / 1e12 * 5.0
}

// String returns a human-readable cost summary.
func (ce *CostEstimate) String() string {
	if ce.CacheHit {
		return "Cache hit — no cost"
	}
	return fmt.Sprintf("~$%.4f USD (%d bytes billed)", ce.EstimatedCostUSD(), ce.TotalBytesBilled)
}
