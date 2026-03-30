package middleware

import (
	"bufio"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
)

// RequestMetrics tracks in-memory HTTP request metrics for the stats endpoint.
// All operations are lock-free for reads of counters (atomic) and use a mutex
// only when recording latency samples or reading snapshots.
type RequestMetrics struct {
	totalRequests  atomic.Int64
	totalErrors    atomic.Int64 // status >= 500
	total4xx       atomic.Int64
	startTime      time.Time

	mu             sync.Mutex
	latencySamples []int64 // nanoseconds, circular buffer
	sampleIdx      int
	sampleCount    int
	maxSamples     int

	statusCounts   sync.Map // status code bucket string -> *atomic.Int64
	statusDetail   sync.Map // exact status code string -> *atomic.Int64
	methodCounts   sync.Map // method string -> *atomic.Int64
	topEndpoints   sync.Map // route pattern -> *endpointStats
}

// NewRequestMetrics creates a new RequestMetrics collector.
func NewRequestMetrics() *RequestMetrics {
	return &RequestMetrics{
		startTime:      time.Now(),
		latencySamples: make([]int64, 10000),
		maxSamples:     10000,
	}
}

// RequestMetricsSnapshot is a point-in-time snapshot of request metrics.
type RequestMetricsSnapshot struct {
	TotalRequests  int64              `json:"totalRequests"`
	TotalErrors    int64              `json:"totalErrors"`
	Total4xx       int64              `json:"total4xx"`
	ErrorRate      float64            `json:"errorRate"`
	UptimeSeconds  int64              `json:"uptimeSeconds"`
	RequestsPerSec float64            `json:"requestsPerSec"`
	LatencyP50Ms   float64            `json:"latencyP50Ms"`
	LatencyP95Ms   float64            `json:"latencyP95Ms"`
	LatencyP99Ms   float64            `json:"latencyP99Ms"`
	StatusCodes    map[string]int64   `json:"statusCodes"`
	StatusDetail   map[string]int64   `json:"statusDetail"`
	MethodCounts   map[string]int64   `json:"methodCounts"`
	TopEndpoints   []EndpointSnapshot `json:"topEndpoints"`
}

// EndpointSnapshot is a per-route summary.
type EndpointSnapshot struct {
	Route      string  `json:"route"`
	Count      int64   `json:"count"`
	AvgLatency float64 `json:"avgLatencyMs"`
	Errors     int64   `json:"errors"`
}

type endpointStats struct {
	count      atomic.Int64
	totalNs    atomic.Int64
	errors     atomic.Int64
}

// Middleware returns an http.Handler middleware that records request metrics.
func (rm *RequestMetrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(ww, r)

		duration := time.Since(start).Nanoseconds()
		status := ww.status

		rm.totalRequests.Add(1)
		if status >= 500 {
			rm.totalErrors.Add(1)
		}
		if status >= 400 && status < 500 {
			rm.total4xx.Add(1)
		}

		// Record latency sample
		rm.mu.Lock()
		rm.latencySamples[rm.sampleIdx] = duration
		rm.sampleIdx = (rm.sampleIdx + 1) % rm.maxSamples
		if rm.sampleCount < rm.maxSamples {
			rm.sampleCount++
		}
		rm.mu.Unlock()

		// Increment status code counter (bucket: 2xx, 4xx, 5xx)
		statusKey := statusBucket(status)
		counter, _ := rm.statusCounts.LoadOrStore(statusKey, &atomic.Int64{})
		counter.(*atomic.Int64).Add(1)

		// Increment granular status counter (exact code: 200, 404, 500, etc.)
		detailKey := strconv.Itoa(status)
		detailCounter, _ := rm.statusDetail.LoadOrStore(detailKey, &atomic.Int64{})
		detailCounter.(*atomic.Int64).Add(1)

		// Increment method counter
		method := strings.ToUpper(r.Method)
		mCounter, _ := rm.methodCounts.LoadOrStore(method, &atomic.Int64{})
		mCounter.(*atomic.Int64).Add(1)

		// Track per-endpoint stats
		rctx := chi.RouteContext(r.Context())
		if rctx != nil && rctx.RoutePattern() != "" {
			route := method + " " + rctx.RoutePattern()
			ep, _ := rm.topEndpoints.LoadOrStore(route, &endpointStats{})
			es := ep.(*endpointStats)
			es.count.Add(1)
			es.totalNs.Add(duration)
			if status >= 500 {
				es.errors.Add(1)
			}
		}
	})
}

// Snapshot returns a point-in-time view of the metrics.
func (rm *RequestMetrics) Snapshot() RequestMetricsSnapshot {
	total := rm.totalRequests.Load()
	errors := rm.totalErrors.Load()
	errors4xx := rm.total4xx.Load()
	uptime := int64(time.Since(rm.startTime).Seconds())

	var rps float64
	if uptime > 0 {
		rps = float64(total) / float64(uptime)
	}

	var errRate float64
	if total > 0 {
		errRate = float64(errors) / float64(total)
	}

	// Copy and sort latency samples for percentile calculation
	rm.mu.Lock()
	n := rm.sampleCount
	samples := make([]int64, n)
	if n == rm.maxSamples {
		copy(samples, rm.latencySamples)
	} else {
		copy(samples, rm.latencySamples[:n])
	}
	rm.mu.Unlock()

	var p50, p95, p99 float64
	if n > 0 {
		sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
		p50 = float64(samples[percentileIdx(n, 50)]) / 1e6
		p95 = float64(samples[percentileIdx(n, 95)]) / 1e6
		p99 = float64(samples[percentileIdx(n, 99)]) / 1e6
	}

	statusCodes := make(map[string]int64)
	rm.statusCounts.Range(func(key, value any) bool {
		statusCodes[key.(string)] = value.(*atomic.Int64).Load()
		return true
	})

	statusDetail := make(map[string]int64)
	rm.statusDetail.Range(func(key, value any) bool {
		statusDetail[key.(string)] = value.(*atomic.Int64).Load()
		return true
	})

	methodCounts := make(map[string]int64)
	rm.methodCounts.Range(func(key, value any) bool {
		methodCounts[key.(string)] = value.(*atomic.Int64).Load()
		return true
	})

	var endpoints []EndpointSnapshot
	rm.topEndpoints.Range(func(key, value any) bool {
		es := value.(*endpointStats)
		c := es.count.Load()
		var avg float64
		if c > 0 {
			avg = float64(es.totalNs.Load()) / float64(c) / 1e6
		}
		endpoints = append(endpoints, EndpointSnapshot{
			Route:      key.(string),
			Count:      c,
			AvgLatency: avg,
			Errors:     es.errors.Load(),
		})
		return true
	})
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].Count > endpoints[j].Count
	})
	if len(endpoints) > 20 {
		endpoints = endpoints[:20]
	}

	return RequestMetricsSnapshot{
		TotalRequests:  total,
		TotalErrors:    errors,
		Total4xx:       errors4xx,
		ErrorRate:      errRate,
		UptimeSeconds:  uptime,
		RequestsPerSec: rps,
		LatencyP50Ms:   p50,
		LatencyP95Ms:   p95,
		LatencyP99Ms:   p99,
		StatusCodes:    statusCodes,
		StatusDetail:   statusDetail,
		MethodCounts:   methodCounts,
		TopEndpoints:   endpoints,
	}
}

func percentileIdx(n, pct int) int {
	idx := (n * pct / 100)
	if idx >= n {
		idx = n - 1
	}
	return idx
}

func statusBucket(status int) string {
	switch {
	case status < 200:
		return "1xx"
	case status < 300:
		return "2xx"
	case status < 400:
		return "3xx"
	case status < 500:
		return "4xx"
	default:
		return "5xx"
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.wroteHeader = true
	}
	return rw.ResponseWriter.Write(b)
}

// Unwrap returns the underlying ResponseWriter for middleware compatibility.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// Hijack implements http.Hijacker so WebSocket upgrades work through this wrapper.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Flush implements http.Flusher for streaming responses.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
