package tcp

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestCheckSuccess(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test listener: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)
	result := Check(context.Background(), "127.0.0.1", addr.Port, time.Second)
	if !result.Success {
		t.Errorf("Check succeeded expected, got Success=false, Error=%q", result.Error)
	}
}

func TestCheckConnectionRefused(t *testing.T) {
	// Bind and immediately close - the port is very likely free right
	// after, and nothing is listening on it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find a free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	result := Check(context.Background(), "127.0.0.1", port, time.Second)
	if result.Success {
		t.Error("Check succeeded against a closed port, want failure")
	}
	if result.Error == "" {
		t.Error("Check failure should set a non-empty Error")
	}
}

func TestCheckTimeout(t *testing.T) {
	// 192.0.2.0/24 is TEST-NET-1 (RFC 5737) - guaranteed non-routable, so
	// the dial will hang until our timeout fires rather than failing
	// immediately.
	result := Check(context.Background(), "192.0.2.1", 9, 100*time.Millisecond)
	if result.Success {
		t.Error("Check succeeded against a non-routable address, want failure")
	}
}
