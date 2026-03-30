package jwt

import (
	"testing"
	"time"
)

func TestCreateAndValidate_RoundTrip(t *testing.T) {
	mgr := NewManager("test-secret-key-long-enough", 15*time.Minute, 7*24*time.Hour)

	token, err := mgr.CreateAccessToken("user-123", "user@example.com")
	if err != nil {
		t.Fatalf("CreateAccessToken: %v", err)
	}

	claims, err := mgr.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
	}
	if claims.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "user@example.com")
	}
	if claims.Subject != "user-123" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "user-123")
	}
}

func TestValidate_WrongSecret(t *testing.T) {
	mgr1 := NewManager("secret-one", 15*time.Minute, 7*24*time.Hour)
	mgr2 := NewManager("secret-two", 15*time.Minute, 7*24*time.Hour)

	token, err := mgr1.CreateAccessToken("user-1", "a@b.com")
	if err != nil {
		t.Fatalf("CreateAccessToken: %v", err)
	}

	_, err = mgr2.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("expected error validating token with wrong secret")
	}
}

func TestValidate_GarbageToken(t *testing.T) {
	mgr := NewManager("secret", 15*time.Minute, 7*24*time.Hour)

	_, err := mgr.ValidateAccessToken("not-a-jwt")
	if err == nil {
		t.Fatal("expected error for garbage token")
	}
}

func TestValidate_EmptyToken(t *testing.T) {
	mgr := NewManager("secret", 15*time.Minute, 7*24*time.Hour)

	_, err := mgr.ValidateAccessToken("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestGenerateRefreshToken_Unique(t *testing.T) {
	tokens := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		tok, err := GenerateRefreshToken()
		if err != nil {
			t.Fatalf("GenerateRefreshToken: %v", err)
		}
		if len(tok) == 0 {
			t.Fatal("empty refresh token")
		}
		if _, exists := tokens[tok]; exists {
			t.Fatalf("duplicate refresh token on iteration %d", i)
		}
		tokens[tok] = struct{}{}
	}
}

func TestRefreshExpiry(t *testing.T) {
	mgr := NewManager("secret", 15*time.Minute, 30*24*time.Hour)
	if mgr.RefreshExpiry() != 30*24*time.Hour {
		t.Errorf("RefreshExpiry = %v, want 30 days", mgr.RefreshExpiry())
	}
}
