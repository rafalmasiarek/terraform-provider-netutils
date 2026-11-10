package tcp

import (
	"context"
	"fmt"
	"net"
	"time"
)

// Result is the outcome of a single TCP connectivity check.
type Result struct {
	Success    bool
	Error      string
	DurationMS int64
}

// Check attempts a TCP connection to host:port, bounded by timeout. It
// never returns a Go error - connection failure is reported in the
// Result itself, since "couldn't connect" is an expected, meaningful
// outcome for a diagnostic check, not an exceptional one.
func Check(ctx context.Context, host string, port int, timeout time.Duration) Result {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	duration := time.Since(start)

	if err != nil {
		return Result{Success: false, Error: err.Error(), DurationMS: duration.Milliseconds()}
	}
	defer conn.Close()

	return Result{Success: true, DurationMS: duration.Milliseconds()}
}
