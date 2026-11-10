package httpcheck

import (
	"context"
	"net/http"
	"time"
)

// Result is the outcome of a single HTTP check.
type Result struct {
	Success    bool
	StatusCode int // 0 if the request never got a response (connection failure)
	Error      string
	DurationMS int64
}

// Check performs an HTTP GET against url, bounded by timeout. If
// expectedStatus is non-zero, success requires an exact match; otherwise
// success is any 2xx or 3xx status. Like tcp.Check, connection and
// transport failures are reported in the Result, not as a Go error.
func Check(ctx context.Context, url string, timeout time.Duration, expectedStatus int) Result {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{Success: false, Error: err.Error()}
	}

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)
	if err != nil {
		return Result{Success: false, Error: err.Error(), DurationMS: duration.Milliseconds()}
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= 200 && resp.StatusCode < 400
	if expectedStatus != 0 {
		success = resp.StatusCode == expectedStatus
	}

	return Result{Success: success, StatusCode: resp.StatusCode, DurationMS: duration.Milliseconds()}
}
