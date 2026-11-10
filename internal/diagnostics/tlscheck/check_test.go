package tlscheck

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestCheckSuccess(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("failed to split test server host:port: %v", err)
	}
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	result := Check(context.Background(), host, port, time.Second)
	if !result.Success {
		t.Fatalf("Check failed: %s", result.Error)
	}
	if result.Expired {
		t.Error("a freshly generated test certificate should not be reported as expired")
	}
	if result.DaysUntilExpiry <= 0 {
		t.Errorf("DaysUntilExpiry = %d, want > 0 for a freshly generated test certificate", result.DaysUntilExpiry)
	}
	if result.NotBefore.IsZero() || result.NotAfter.IsZero() {
		t.Error("NotBefore/NotAfter should be populated")
	}
}

func TestCheckConnectionFailure(t *testing.T) {
	result := Check(context.Background(), "127.0.0.1", 1, time.Second)
	if result.Success {
		t.Error("Check against an unreachable address should fail")
	}
	if result.Error == "" {
		t.Error("Check failure should set a non-empty Error")
	}
}

func TestCheckNonTLSPort(t *testing.T) {
	// A plain TCP listener that never speaks TLS - the handshake itself
	// should fail cleanly rather than hang or panic.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test listener: %v", err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Just hold the connection open without speaking TLS, so the
		// client's handshake read eventually fails/times out cleanly.
		buf := make([]byte, 1)
		conn.Read(buf)
	}()

	port := ln.Addr().(*net.TCPAddr).Port
	result := Check(context.Background(), "127.0.0.1", port, 500*time.Millisecond)
	if result.Success {
		t.Error("Check against a non-TLS listener should fail")
	}
}
