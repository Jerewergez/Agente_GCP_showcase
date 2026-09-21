package bq

import (
	"context"
	"fmt"

	"google.golang.org/api/iterator"
)

// TableInfo holds summary information about a BigQuery table.
type TableInfo struct {
	ProjectID string `json:"projectId"`
	DatasetID string `json:"datasetId"`
	TableID   string `json:"tableId"`
	Type      string `json:"type"`
	Location  string `json:"location,omitempty"`
}

// ListTables returns all tables in the specified dataset.
// The datasetID is expected in the form "project.dataset" or just
// "dataset" (in which case the client's default project is used).
func (c *Client) ListTables(ctx context.Context, datasetID string) ([]TableInfo, error) {
	ds := c.client.Dataset(datasetID)
	metadata, err := ds.Metadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("dataset metadata for %q: %w", datasetID, err)
	}

	tableIter := ds.Tables(ctx)
	var tables []TableInfo
	for {
		t, err := tableIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterate tables in %q: %w", datasetID, err)
		}

		tm, err := t.Metadata(ctx)
		if err != nil {
			// skip tables we can't read metadata for
			tables = append(tables, TableInfo{
				ProjectID: c.ProjectID,
				DatasetID: datasetID,
				TableID:   t.TableID,
				Type:      "unknown",
			})
			continue
		}

		tables = append(tables, TableInfo{
			ProjectID: c.ProjectID,
			DatasetID: datasetID,
			TableID:   t.TableID,
			Type:      string(tm.Type),
			Location:  metadata.Location,
		})
	}

	if tables == nil {
		tables = []TableInfo{} // always return an empty slice, not nil
	}
	return tables, nil
}

// ListAvailableTables is a convenience that lists both SILVER and GOLD
// tables using the configured dataset env vars.
func (c *Client) ListAvailableTables(ctx context.Context) (map[string][]TableInfo, error) {
	result := make(map[string][]TableInfo)

	silver := ctx.Value("silverDataset")
	if silver == nil {
		silver = osGetenv("BQ_DATASET_SILVER")
	}
	if s, ok := silver.(string); ok && s != "" {
		tables, err := c.ListTables(ctx, s)
		if err != nil {
			return nil, fmt.Errorf("list silver tables: %w", err)
		}
		result["SILVER"] = tables
	}

	gold := ctx.Value("goldDataset")
	if gold == nil {
		gold = osGetenv("BQ_DATASET_GOLD")
	}
	if g, ok := gold.(string); ok && g != "" {
		tables, err := c.ListTables(ctx, g)
		if err != nil {
			return nil, fmt.Errorf("list gold tables: %w", err)
		}
		result["GOLD"] = tables
	}

	return result, nil
}

// osGetenv is a package-level variable so tests can override it.
var osGetenv = func(key string) string {
	// Using context values for dataset names would be cleaner;
	// this function exists to decouple from os.Getenv in tests.
	return ""
}
