package models

import (
	"testing"
	"time"
)

func TestShareLink_IsExpired_NilExpiry(t *testing.T) {
	link := &ShareLink{ExpiresAt: nil}
	if link.IsExpired() {
		t.Error("expected not expired when ExpiresAt is nil")
	}
}

func TestShareLink_IsExpired_FutureExpiry(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	link := &ShareLink{ExpiresAt: &future}
	if link.IsExpired() {
		t.Error("expected not expired for future time")
	}
}

func TestShareLink_IsExpired_PastExpiry(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	link := &ShareLink{ExpiresAt: &past}
	if !link.IsExpired() {
		t.Error("expected expired for past time")
	}
}

func TestShareLink_IsExpired_JustExpired(t *testing.T) {
	past := time.Now().Add(-1 * time.Millisecond)
	link := &ShareLink{ExpiresAt: &past}
	if !link.IsExpired() {
		t.Error("expected expired for time just past")
	}
}
