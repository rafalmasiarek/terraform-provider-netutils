package httpcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckDefaultSuccessRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	result := Check(context.Background(), server.URL, time.Second, 0)
	if !result.Success || result.StatusCode != 200 {
		t.Errorf("Check(default) = %+v, want Success=true StatusCode=200", result)
	}
}

func TestCheckExpectedStatusMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result := Check(context.Background(), server.URL, time.Second, 200)
	if result.Success {
		t.Errorf("Check(expected=200) against a 404 response should fail, got %+v", result)
	}
	if result.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", result.StatusCode)
	}
}

func TestCheckExpectedStatusMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result := Check(context.Background(), server.URL, time.Second, 404)
	if !result.Success {
		t.Errorf("Check(expected=404) against a 404 response should succeed, got %+v", result)
	}
}

func TestCheckDefaultRangeRejects5xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	result := Check(context.Background(), server.URL, time.Second, 0)
	if result.Success {
		t.Errorf("Check(default) against a 500 response should fail, got %+v", result)
	}
}

func TestCheckConnectionFailure(t *testing.T) {
	result := Check(context.Background(), "http://127.0.0.1:1", time.Second, 0)
	if result.Success {
		t.Error("Check against an unreachable address should fail")
	}
	if result.Error == "" {
		t.Error("Check failure should set a non-empty Error")
	}
}
