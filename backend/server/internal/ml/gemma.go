// Package ml provides clients for machine-learning model inference,
// including the Gemma Cloud Run endpoint for NL→SQL fallback.
package ml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// ---------------------------------------------------------------------------
// GemmaClient — HTTP client for Gemma on Cloud Run
// ---------------------------------------------------------------------------

// GemmaRequest is the payload sent to the Gemma endpoint.
type GemmaRequest struct {
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// GemmaResponse is the expected response from the Gemma endpoint.
type GemmaResponse struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence,omitempty"`
}

// GemmaClient sends inference requests to a Gemma model hosted on
// Cloud Run. Set GEMMA_URL env var to enable; if unset, all calls
// return an ErrGemmaNotConfigured error.
type GemmaClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// ErrGemmaNotConfigured is returned when GEMMA_URL is not set.
var ErrGemmaNotConfigured = fmt.Errorf("gemma: GEMMA_URL not set")

// NewGemmaClient creates a Gemma client from environment config.
// If GEMMA_URL is empty the client will return ErrGemmaNotConfigured
// on every request, acting as a graceful fallback.
func NewGemmaClient() *GemmaClient {
	url := os.Getenv("GEMMA_URL")
	return &GemmaClient{
		baseURL: url,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 30 * time.Second,
	}
}

// NewGemmaClientWithURL creates a Gemma client with an explicit URL.
func NewGemmaClientWithURL(url string, timeout time.Duration) *GemmaClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &GemmaClient{
		baseURL: url,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// IsConfigured returns true if a Gemma endpoint URL has been set.
func (gc *GemmaClient) IsConfigured() bool {
	return gc.baseURL != ""
}

// Predict sends a prompt to the Gemma Cloud Run endpoint and returns
// the generated text with an optional confidence score.
func (gc *GemmaClient) Predict(ctx context.Context, prompt string, opts ...PredictOption) (*GemmaResponse, error) {
	if gc.baseURL == "" {
		return nil, ErrGemmaNotConfigured
	}

	req := GemmaRequest{
		Prompt:      prompt,
		MaxTokens:   512,
		Temperature: 0.3,
	}
	for _, o := range opts {
		o(&req)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("gemma marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, gc.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemma create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := gc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemma POST: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gemma read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemma HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var gemmaResp GemmaResponse
	if err := json.Unmarshal(respBody, &gemmaResp); err != nil {
		return nil, fmt.Errorf("gemma unmarshal response: %w", err)
	}

	if gemmaResp.Text == "" {
		return nil, fmt.Errorf("gemma returned empty text")
	}

	return &gemmaResp, nil
}

// PredictOption allows customising Gemma inference parameters.
type PredictOption func(*GemmaRequest)

// WithMaxTokens sets the maximum number of tokens in the response.
func WithMaxTokens(n int) PredictOption {
	return func(r *GemmaRequest) {
		r.MaxTokens = n
	}
}

// WithTemperature sets the generation temperature.
func WithTemperature(t float64) PredictOption {
	return func(r *GemmaRequest) {
		r.Temperature = t
	}
}
