// Package bq provides a BigQuery client initialised from a service
// account key path, with safe query builder, cost estimation, and
// table listing utilities.
package bq

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"
)

// Client wraps a BigQuery client and the configured project/dataset.
type Client struct {
	ProjectID string
	client    *bigquery.Client
}

// NewClient creates a BigQuery client from the GCP_SA_KEY_PATH env var.
// The env var must point to a valid JSON service account key file.
func NewClient(ctx context.Context) (*Client, error) {
	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		return nil, fmt.Errorf("GCP_PROJECT env var not set")
	}

	saKeyPath := os.Getenv("GCP_SA_KEY_PATH")
	if saKeyPath == "" {
		return nil, fmt.Errorf("GCP_SA_KEY_PATH env var not set")
	}

	client, err := bigquery.NewClient(ctx, projectID, option.WithCredentialsFile(saKeyPath))
	if err != nil {
		return nil, fmt.Errorf("bigquery.NewClient: %w", err)
	}

	log.Printf("bq: connected to project %s using SA key %s", projectID, saKeyPath)
	return &Client{
		ProjectID: projectID,
		client:    client,
	}, nil
}

// Close releases the underlying BigQuery client resources.
func (c *Client) Close() error {
	return c.client.Close()
}
