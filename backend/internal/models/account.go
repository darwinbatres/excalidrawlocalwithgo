package models

import "time"

// Account represents an OAuth2 provider link for a user.
type Account struct {
	ID                string  `json:"id"`
	UserID            string  `json:"userId"`
	Type              string  `json:"type"`
	Provider          string  `json:"provider"`
	ProviderAccountID string  `json:"providerAccountId"`
	RefreshToken      *string `json:"-"`
	AccessToken       *string `json:"-"`
	ExpiresAt         *int    `json:"expiresAt,omitempty"`
	TokenType         *string `json:"tokenType,omitempty"`
	Scope             *string `json:"scope,omitempty"`
	IDToken           *string `json:"-"`
	SessionState      *string `json:"sessionState,omitempty"`
}

// RefreshToken represents a JWT refresh token stored for revocation/rotation.
type RefreshToken struct {
	ID         string     `json:"id"`
	UserID     string     `json:"userId"`
	TokenHash  string     `json:"-"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
	ReplacedBy *string    `json:"replacedBy,omitempty"`
	UserAgent  *string    `json:"userAgent,omitempty"`
	IP         *string    `json:"ip,omitempty"`
}
