package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBruteForce_AllowsNormalRequests(t *testing.T) {
	bf := NewBruteForce(BruteForceConfig{MaxAttempts: 3})
	handler := bf.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestBruteForce_BlocksAfterMaxFailures(t *testing.T) {
	bf := NewBruteForce(BruteForceConfig{MaxAttempts: 3, Window: time.Minute, Lockout: time.Minute})
	ip := "1.2.3.4:12345"

	bf.RecordFailure(ip)
	bf.RecordFailure(ip)
	locked := bf.RecordFailure(ip)

	if !locked {
		t.Error("expected locked after 3 failures")
	}

	if !bf.IsLocked(ip) {
		t.Error("IsLocked should return true")
	}

	handler := bf.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = ip
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}

	if ra := w.Header().Get("Retry-After"); ra == "" {
		t.Error("expected Retry-After header")
	}
}

func TestBruteForce_RecordSuccessResetsCount(t *testing.T) {
	bf := NewBruteForce(BruteForceConfig{MaxAttempts: 3})
	ip := "5.6.7.8:9999"

	bf.RecordFailure(ip)
	bf.RecordFailure(ip)
	bf.RecordSuccess(ip)

	// Should be reset — third failure should NOT lock out
	locked := bf.RecordFailure(ip)
	if locked {
		t.Error("should not be locked after success reset")
	}
}

func TestBruteForce_DifferentIPsIndependent(t *testing.T) {
	bf := NewBruteForce(BruteForceConfig{MaxAttempts: 2})

	bf.RecordFailure("10.0.0.1:1111")
	bf.RecordFailure("10.0.0.1:1111")

	if !bf.IsLocked("10.0.0.1:1111") {
		t.Error("IP1 should be locked")
	}
	if bf.IsLocked("10.0.0.2:2222") {
		t.Error("IP2 should not be locked")
	}
}

func TestBruteForce_WindowExpiry(t *testing.T) {
	bf := NewBruteForce(BruteForceConfig{
		MaxAttempts: 5,
		Window:      50 * time.Millisecond,
		Lockout:     50 * time.Millisecond,
	})
	ip := "11.22.33.44:5555"

	for i := 0; i < 5; i++ {
		bf.RecordFailure(ip)
	}

	if !bf.IsLocked(ip) {
		t.Error("should be locked immediately after max failures")
	}

	// Wait for lockout to expire
	time.Sleep(100 * time.Millisecond)

	if bf.IsLocked(ip) {
		t.Error("lockout should have expired")
	}
}

func TestBruteForce_Evict(t *testing.T) {
	bf := NewBruteForce(BruteForceConfig{
		MaxAttempts: 2,
		Window:      10 * time.Millisecond,
		Lockout:     10 * time.Millisecond,
	})
	ip := "99.99.99.99:1234"

	bf.RecordFailure(ip)
	bf.RecordFailure(ip)

	time.Sleep(50 * time.Millisecond)
	bf.evict()

	bf.mu.Lock()
	_, exists := bf.attempts[ip]
	bf.mu.Unlock()

	if exists {
		t.Error("expected evict to remove expired entry")
	}
}

func TestBruteForce_DefaultConfig(t *testing.T) {
	bf := NewBruteForce(BruteForceConfig{})

	if bf.maxAttempts != 5 {
		t.Errorf("expected default maxAttempts=5, got %d", bf.maxAttempts)
	}
	if bf.window != 15*time.Minute {
		t.Errorf("expected default window=15m, got %v", bf.window)
	}
	if bf.lockout != 15*time.Minute {
		t.Errorf("expected default lockout=15m, got %v", bf.lockout)
	}
}
