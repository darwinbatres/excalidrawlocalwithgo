package models

import "time"

// ShareLink represents a shareable link for read-only board access.
type ShareLink struct {
	ID        string     `json:"id"`
	BoardID   string     `json:"boardId"`
	Token     string     `json:"token"`
	Role      BoardRole  `json:"role"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	CreatedBy string     `json:"createdBy"`
}

// IsExpired returns true if the share link has passed its expiration time.
func (s *ShareLink) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}
