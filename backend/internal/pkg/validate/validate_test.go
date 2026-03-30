package validate

import (
	"net/http"
	"strings"
	"testing"
)

type testStruct struct {
	Email    string  `json:"email" validate:"required,email,max=255"`
	Password string  `json:"password" validate:"required,min=8,max=128"`
	Name     *string `json:"name" validate:"omitempty,min=1,max=100"`
}

func TestStruct_Valid(t *testing.T) {
	s := testStruct{Email: "user@example.com", Password: "password123"}
	if apiErr := Struct(s); apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
}

func TestStruct_MissingRequired(t *testing.T) {
	s := testStruct{Email: "", Password: "password123"}
	apiErr := Struct(s)
	if apiErr == nil {
		t.Fatal("expected validation error for missing email")
	}
	if apiErr.Status != 422 {
		t.Errorf("Status = %d, want 422", apiErr.Status)
	}
}

func TestStruct_InvalidEmail(t *testing.T) {
	s := testStruct{Email: "not-an-email", Password: "password123"}
	apiErr := Struct(s)
	if apiErr == nil {
		t.Fatal("expected validation error for invalid email")
	}
}

func TestStruct_PasswordTooShort(t *testing.T) {
	s := testStruct{Email: "a@b.com", Password: "short"}
	apiErr := Struct(s)
	if apiErr == nil {
		t.Fatal("expected validation error for short password")
	}
}

type slugStruct struct {
	Slug string `validate:"required,slug"`
}

func TestSlugValidation(t *testing.T) {
	tests := []struct {
		name  string
		slug  string
		valid bool
	}{
		{"valid simple", "my-org", true},
		{"valid single", "test", true},
		{"valid with numbers", "org-123", true},
		{"invalid uppercase", "My-Org", false},
		{"invalid spaces", "my org", false},
		{"invalid leading hyphen", "-org", false},
		{"invalid trailing hyphen", "org-", false},
		{"invalid empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := Struct(slugStruct{Slug: tt.slug})
			if tt.valid && apiErr != nil {
				t.Errorf("expected slug %q to be valid, got error", tt.slug)
			}
			if !tt.valid && apiErr == nil {
				t.Errorf("expected slug %q to be invalid", tt.slug)
			}
		})
	}
}

func TestDecodeAndValidate_Valid(t *testing.T) {
	body := `{"email":"a@b.com","password":"password123"}`
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var s testStruct
	if apiErr := DecodeAndValidate(req, &s); apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if s.Email != "a@b.com" {
		t.Errorf("Email = %q, want %q", s.Email, "a@b.com")
	}
}

func TestDecodeAndValidate_InvalidJSON(t *testing.T) {
	body := `{bad json}`
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	var s testStruct
	apiErr := DecodeAndValidate(req, &s)
	if apiErr == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestDecodeAndValidate_ValidJSON_FailsValidation(t *testing.T) {
	body := `{"email":"not-email","password":"x"}`
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	var s testStruct
	apiErr := DecodeAndValidate(req, &s)
	if apiErr == nil {
		t.Fatal("expected validation error")
	}
}

type strongPwStruct struct {
	Password string `validate:"required,min=8,strongpassword"`
}

func TestStrongPasswordValidation(t *testing.T) {
	tests := []struct {
		name  string
		pw    string
		valid bool
	}{
		{"valid upper+digit", "Password1", true},
		{"valid upper+special", "Password!", true},
		{"missing uppercase", "password1", false},
		{"missing digit/special", "Password", false},
		{"all lowercase no digit", "abcdefgh", false},
		{"upper+digit+special", "P@ssw0rd!", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := Struct(strongPwStruct{Password: tt.pw})
			if tt.valid && apiErr != nil {
				t.Errorf("expected password %q to be valid, got error", tt.pw)
			}
			if !tt.valid && apiErr == nil {
				t.Errorf("expected password %q to be invalid", tt.pw)
			}
		})
	}
}
