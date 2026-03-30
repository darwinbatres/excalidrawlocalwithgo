package middleware

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/response"
)

// BruteForce tracks failed authentication attempts per IP and applies
// progressive delays. After maxAttempts failures within the window,
// further requests are rejected until the lockout expires.
type BruteForce struct {
	mu          sync.Mutex
	attempts    map[string]*attemptRecord
	maxAttempts int
	window      time.Duration
	lockout     time.Duration
}

type attemptRecord struct {
	count   int
	firstAt time.Time
	lockedUntil time.Time
}

// BruteForceConfig configures the brute-force protector.
type BruteForceConfig struct {
	MaxAttempts int           // failures before lockout (default 5)
	Window      time.Duration // time window for counting failures (default 15m)
	Lockout     time.Duration // lockout duration after max failures (default 15m)
}

// NewBruteForce creates a brute-force protector. Call StartCleanup to
// run periodic eviction of stale entries.
func NewBruteForce(cfg BruteForceConfig) *BruteForce {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 5
	}
	if cfg.Window <= 0 {
		cfg.Window = 15 * time.Minute
	}
	if cfg.Lockout <= 0 {
		cfg.Lockout = 15 * time.Minute
	}
	return &BruteForce{
		attempts:    make(map[string]*attemptRecord),
		maxAttempts: cfg.MaxAttempts,
		window:      cfg.Window,
		lockout:     cfg.Lockout,
	}
}

// StartCleanup runs a background goroutine that evicts expired entries.
// Stops when the provided done channel is closed.
func (bf *BruteForce) StartCleanup(done <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(bf.window)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				bf.evict()
			}
		}
	}()
}

// Middleware returns an HTTP middleware that blocks requests from
// IPs that have exceeded the failure threshold.
func (bf *BruteForce) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r.RemoteAddr)

			bf.mu.Lock()
			rec := bf.attempts[ip]
			if rec != nil {
				now := time.Now()
				// Check lockout
				if !rec.lockedUntil.IsZero() && now.Before(rec.lockedUntil) {
					bf.mu.Unlock()
					retryAfter := int(time.Until(rec.lockedUntil).Seconds()) + 1
					w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
					response.Err(w, r, apierror.ErrTooManyRequests.WithMessage("Too many failed attempts. Try again later."))
					return
				}
				// Reset if window expired
				if now.Sub(rec.firstAt) > bf.window {
					rec.count = 0
					rec.firstAt = now
					rec.lockedUntil = time.Time{}
				}
			}
			bf.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

// RecordFailure records a failed authentication attempt for the given IP.
// Returns true if the IP is now locked out.
func (bf *BruteForce) RecordFailure(ip string) bool {
	ip = extractIP(ip)
	bf.mu.Lock()
	defer bf.mu.Unlock()

	now := time.Now()
	rec, ok := bf.attempts[ip]
	if !ok {
		bf.attempts[ip] = &attemptRecord{count: 1, firstAt: now}
		return false
	}

	// Reset if window expired
	if now.Sub(rec.firstAt) > bf.window {
		rec.count = 1
		rec.firstAt = now
		rec.lockedUntil = time.Time{}
		return false
	}

	rec.count++
	if rec.count >= bf.maxAttempts {
		rec.lockedUntil = now.Add(bf.lockout)
		return true
	}
	return false
}

// RecordSuccess clears the failure count for the given IP.
func (bf *BruteForce) RecordSuccess(ip string) {
	ip = extractIP(ip)
	bf.mu.Lock()
	defer bf.mu.Unlock()
	delete(bf.attempts, ip)
}

// IsLocked returns true if the given IP is currently locked out.
func (bf *BruteForce) IsLocked(ip string) bool {
	ip = extractIP(ip)
	bf.mu.Lock()
	defer bf.mu.Unlock()
	rec, ok := bf.attempts[ip]
	if !ok {
		return false
	}
	return !rec.lockedUntil.IsZero() && time.Now().Before(rec.lockedUntil)
}

// BruteForceStats holds a snapshot of brute-force protector state.
type BruteForceStats struct {
	TrackedIPs int `json:"trackedIPs"`
	LockedIPs  int `json:"lockedIPs"`
}

// Stats returns a snapshot of brute-force tracking state.
func (bf *BruteForce) Stats() BruteForceStats {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	now := time.Now()
	locked := 0
	for _, rec := range bf.attempts {
		if !rec.lockedUntil.IsZero() && now.Before(rec.lockedUntil) {
			locked++
		}
	}
	return BruteForceStats{
		TrackedIPs: len(bf.attempts),
		LockedIPs:  locked,
	}
}

func (bf *BruteForce) evict() {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	now := time.Now()
	for ip, rec := range bf.attempts {
		expired := now.Sub(rec.firstAt) > bf.window
		lockoutPassed := rec.lockedUntil.IsZero() || now.After(rec.lockedUntil)
		if expired && lockoutPassed {
			delete(bf.attempts, ip)
		}
	}
}

// extractIP strips the port from an address string, returning just the IP.
// Handles both "ip:port" and bare "ip" formats (including IPv6).
func extractIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr // already just an IP
	}
	return host
}

// ExtractIP is the exported version of extractIP for use by handlers.
func ExtractIP(addr string) string {
	return extractIP(addr)
}
