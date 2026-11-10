package iplookup

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

const sampleResponse = `{
  "status": "success",
  "code": 200,
  "message": "OK",
  "data": {
    "ip": "8.8.8.8",
    "hostname": "dns.google",
    "network": "8.8.8.0/24",
    "asn": {"asn": "AS15169", "name": "Google LLC", "type": "content"},
    "location": {"country": "US", "country_name": "United States", "region": null, "city": null, "is_eu": false}
  },
  "errors": [],
  "meta": {}
}`

func TestClientLookupSuccess(t *testing.T) {
	var gotPath, gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, sampleResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL, 2*time.Second)
	resp, err := client.Lookup(context.Background(), LookupOptions{IP: "8.8.8.8", ASN: true, Location: true})
	if err != nil {
		t.Fatalf("Lookup returned error: %v", err)
	}

	if resp.Data.IP != "8.8.8.8" {
		t.Errorf("Data.IP = %q, want 8.8.8.8", resp.Data.IP)
	}
	if resp.Data.ASN == nil || resp.Data.ASN.Name != "Google LLC" {
		t.Errorf("Data.ASN = %+v, want Name=Google LLC", resp.Data.ASN)
	}
	if resp.Data.Location == nil || resp.Data.Location.Country != "US" {
		t.Errorf("Data.Location = %+v, want Country=US", resp.Data.Location)
	}

	if gotPath != "/8.8.8.8" {
		t.Errorf("request path = %q, want /8.8.8.8", gotPath)
	}
	if gotQuery != "abuse=false&asn=true&location=true&rbl=false" {
		t.Errorf("request query = %q, want abuse=false&asn=true&location=true&rbl=false", gotQuery)
	}
}

func TestClientLookupCurrentIP(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, sampleResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL, 2*time.Second)
	if _, err := client.Lookup(context.Background(), LookupOptions{}); err != nil {
		t.Fatalf("Lookup returned error: %v", err)
	}
	if gotPath != "/" {
		t.Errorf("request path = %q, want / (no IP appended)", gotPath)
	}
}

func TestClientLookupMissingOptionalFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"success","code":200,"message":"OK","data":{"ip":"1.2.3.4"},"errors":[],"meta":{}}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, 2*time.Second)
	resp, err := client.Lookup(context.Background(), LookupOptions{IP: "1.2.3.4"})
	if err != nil {
		t.Fatalf("Lookup returned error: %v", err)
	}
	if resp.Data.ASN != nil {
		t.Errorf("Data.ASN = %+v, want nil when the API omits the section", resp.Data.ASN)
	}
	if resp.Data.Location != nil {
		t.Errorf("Data.Location = %+v, want nil when the API omits the section", resp.Data.Location)
	}
}

func TestClientLookupNonRetryable4xx(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "bad request")
	}))
	defer server.Close()

	client := NewClient(server.URL, 2*time.Second)
	client.RetryDelay = time.Millisecond
	_, err := client.Lookup(context.Background(), LookupOptions{IP: "bad-ip"})
	if err == nil {
		t.Fatal("expected an error for a 400 response, got nil")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (a 4xx must not be retried)", calls)
	}
}

func TestClientLookupRetries5xxThenSucceeds(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, "temporarily unavailable")
			return
		}
		fmt.Fprint(w, sampleResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL, 2*time.Second)
	client.RetryDelay = time.Millisecond
	resp, err := client.Lookup(context.Background(), LookupOptions{})
	if err != nil {
		t.Fatalf("Lookup returned error after retries: %v", err)
	}
	if resp.Data.IP != "8.8.8.8" {
		t.Errorf("Data.IP = %q, want 8.8.8.8 after eventual success", resp.Data.IP)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3 (2 failures + 1 success)", calls)
	}
}

func TestClientLookupExhaustsRetriesOn5xx(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "down")
	}))
	defer server.Close()

	client := NewClient(server.URL, 2*time.Second)
	client.RetryDelay = time.Millisecond
	client.MaxRetries = 2
	_, err := client.Lookup(context.Background(), LookupOptions{})
	if err == nil {
		t.Fatal("expected an error after exhausting retries, got nil")
	}
	if calls != 3 { // initial attempt + 2 retries
		t.Errorf("calls = %d, want 3 (1 initial + MaxRetries=2 retries)", calls)
	}
}
