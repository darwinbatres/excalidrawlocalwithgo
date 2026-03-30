package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestMetrics_RecordsRequests(t *testing.T) {
	rm := NewRequestMetrics()

	handler := rm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	snap := rm.Snapshot()
	if snap.TotalRequests != 10 {
		t.Errorf("TotalRequests = %d, want 10", snap.TotalRequests)
	}
	if snap.TotalErrors != 0 {
		t.Errorf("TotalErrors = %d, want 0", snap.TotalErrors)
	}
	if snap.MethodCounts["GET"] != 10 {
		t.Errorf("GET count = %d, want 10", snap.MethodCounts["GET"])
	}
	if snap.StatusCodes["2xx"] != 10 {
		t.Errorf("2xx count = %d, want 10", snap.StatusCodes["2xx"])
	}
}

func TestRequestMetrics_Tracks5xxErrors(t *testing.T) {
	rm := NewRequestMetrics()

	handler := rm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodPost, "/fail", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	snap := rm.Snapshot()
	if snap.TotalErrors != 1 {
		t.Errorf("TotalErrors = %d, want 1", snap.TotalErrors)
	}
	if snap.ErrorRate != 1.0 {
		t.Errorf("ErrorRate = %f, want 1.0", snap.ErrorRate)
	}
	if snap.StatusCodes["5xx"] != 1 {
		t.Errorf("5xx count = %d, want 1", snap.StatusCodes["5xx"])
	}
}

func TestRequestMetrics_Tracks4xxErrors(t *testing.T) {
	rm := NewRequestMetrics()

	handler := rm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	snap := rm.Snapshot()
	if snap.Total4xx != 1 {
		t.Errorf("Total4xx = %d, want 1", snap.Total4xx)
	}
}

func TestRequestMetrics_LatencyPercentiles(t *testing.T) {
	rm := NewRequestMetrics()

	handler := rm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	snap := rm.Snapshot()
	if snap.LatencyP50Ms < 0 {
		t.Errorf("P50 should be >= 0, got %f", snap.LatencyP50Ms)
	}
	if snap.LatencyP95Ms < snap.LatencyP50Ms {
		t.Errorf("P95 (%f) should be >= P50 (%f)", snap.LatencyP95Ms, snap.LatencyP50Ms)
	}
	if snap.LatencyP99Ms < snap.LatencyP95Ms {
		t.Errorf("P99 (%f) should be >= P95 (%f)", snap.LatencyP99Ms, snap.LatencyP95Ms)
	}
}

func TestRequestMetrics_EmptySnapshot(t *testing.T) {
	rm := NewRequestMetrics()
	snap := rm.Snapshot()

	if snap.TotalRequests != 0 {
		t.Errorf("TotalRequests = %d, want 0", snap.TotalRequests)
	}
	if snap.ErrorRate != 0 {
		t.Errorf("ErrorRate = %f, want 0", snap.ErrorRate)
	}
	if snap.LatencyP50Ms != 0 {
		t.Errorf("P50 = %f, want 0", snap.LatencyP50Ms)
	}
}
