package iplookup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the single shared HTTP client used by every IP intelligence
// data source - no data source builds its own HTTP logic.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	MaxRetries int
	RetryDelay time.Duration
}

// NewClient returns a Client with the given base URL and per-request
// timeout. HTTPClient can be swapped afterward for tests (e.g. to point at
// an httptest.Server, or to inject a custom Transport).
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: timeout},
		MaxRetries: 2,
		RetryDelay: 200 * time.Millisecond,
	}
}

// Lookup performs one IP lookup call, retrying transient failures (network
// errors and 5xx responses) up to MaxRetries times before giving up. A 4xx
// response is never retried - it indicates a request-level problem that
// won't change on retry.
func (c *Client) Lookup(ctx context.Context, opts LookupOptions) (*Response, error) {
	u, err := buildURL(c.BaseURL, opts)
	if err != nil {
		return nil, fmt.Errorf("invalid ip_lookup_endpoint or ip: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.RetryDelay):
			}
		}

		resp, err := c.doRequest(ctx, u)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		if httpErr, ok := err.(*HTTPError); ok && httpErr.StatusCode < 500 {
			return nil, err // not transient - retrying won't help
		}
	}

	return nil, fmt.Errorf("ip lookup failed after %d attempts: %w", c.MaxRetries+1, lastErr)
}

func (c *Client) doRequest(ctx context.Context, u string) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	httpResp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, &HTTPError{StatusCode: httpResp.StatusCode, Body: string(body)}
	}

	var parsed Response
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decoding ip lookup response: %w", err)
	}

	return &parsed, nil
}
