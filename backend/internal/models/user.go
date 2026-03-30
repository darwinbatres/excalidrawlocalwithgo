package models

import "time"

// User represents a registered user.
type User struct {
ID            string     `json:"id"`
Email         string     `json:"email"`
EmailVerified *time.Time `json:"emailVerified,omitempty"`
Name          *string    `json:"name,omitempty"`
Image         *string    `json:"image,omitempty"`
PasswordHash  *string    `json:"-"`
CreatedAt     time.Time  `json:"createdAt"`
UpdatedAt     time.Time  `json:"updatedAt"`
}

// UserPublic is the safe representation returned in API responses.
type UserPublic struct {
ID    string  `json:"id"`
Email string  `json:"email"`
Name  *string `json:"name,omitempty"`
Image *string `json:"image,omitempty"`
}

// ToPublic converts a User to its public representation.
func (u *User) ToPublic() UserPublic {
return UserPublic{
ID:    u.ID,
Email: u.Email,
Name:  u.Name,
Image: u.Image,
}
}
