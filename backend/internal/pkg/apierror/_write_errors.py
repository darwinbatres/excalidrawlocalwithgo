#!/usr/bin/env python3
"""Rewrite errors.go with proper formatting and all error constants."""
import os

content = '''package apierror

import (
	"fmt"
	"net/http"
)

// Error is a structured API error with a code, message, HTTP status, and optional details.
type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Status  int            `json:"-"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// New creates a new API error.
func New(status int, code, message string) *Error {
	return &Error{Code: code, Message: message, Status: status}
}

// WithDetails returns a copy of the error with additional details.
func (e *Error) WithDetails(details map[string]any) *Error {
	return &Error{
		Code:    e.Code,
		Message: e.Message,
		Status:  e.Status,
		Details: details,
	}
}

// WithMessage returns a copy of the error with a custom message.
func (e *Error) WithMessage(message string) *Error {
	return &Error{
		Code:    e.Code,
		Message: message,
		Status:  e.Status,
		Details: e.Details,
	}
}

// Pre-defined errors for reuse across handlers.
var (
	// Generic HTTP errors
	ErrBadRequest        = New(http.StatusBadRequest, "BAD_REQUEST", "Invalid request")
	ErrUnauthorized      = New(http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	ErrForbidden         = New(http.StatusForbidden, "FORBIDDEN", "Insufficient permissions")
	ErrNotFound          = New(http.StatusNotFound, "NOT_FOUND", "Resource not found")
	ErrConflict          = New(http.StatusConflict, "CONFLICT", "Resource conflict")
	ErrTooManyRequests   = New(http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests")
	ErrRequestTooLarge   = New(http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "Request body too large")
	ErrInternal          = New(http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred")
	ErrServiceUnavailable = New(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable")

	// Auth errors
	ErrEmailTaken         = New(http.StatusConflict, "EMAIL_TAKEN", "Email already in use")
	ErrUserNotFound       = New(http.StatusNotFound, "USER_NOT_FOUND", "User not found")
	ErrInvalidCredentials = New(http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
	ErrTokenInvalid       = New(http.StatusUnauthorized, "TOKEN_INVALID", "Token is invalid")
	ErrTokenExpired       = New(http.StatusUnauthorized, "TOKEN_EXPIRED", "Token has expired")

	// OAuth errors
	ErrOAuthStateMismatch  = New(http.StatusBadRequest, "OAUTH_STATE_MISMATCH", "OAuth state validation failed")
	ErrOAuthProviderError  = New(http.StatusBadGateway, "OAUTH_PROVIDER_ERROR", "OAuth provider returned an error")
	ErrAccountLinked       = New(http.StatusConflict, "ACCOUNT_LINKED", "Account already linked to another user")

	// Organization errors
	ErrOrgNotFound    = New(http.StatusNotFound, "ORG_NOT_FOUND", "Organization not found")
	ErrSlugTaken      = New(http.StatusConflict, "SLUG_TAKEN", "Organization slug already in use")
	ErrLastOrg        = New(http.StatusConflict, "LAST_ORG", "Cannot delete your only organization")
	ErrOrgHasBoards   = New(http.StatusConflict, "ORG_HAS_BOARDS", "Cannot delete organization with existing boards")

	// Member errors
	ErrMemberNotFound    = New(http.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found")
	ErrAlreadyMember     = New(http.StatusConflict, "ALREADY_MEMBER", "User is already a member of this organization")
	ErrCannotChangeOwner = New(http.StatusForbidden, "CANNOT_CHANGE_OWNER", "Cannot change the owner role")
	ErrCannotRemoveOwner = New(http.StatusForbidden, "CANNOT_REMOVE_OWNER", "Cannot remove the organization owner")

	// Board errors
	ErrBoardNotFound   = New(http.StatusNotFound, "BOARD_NOT_FOUND", "Board not found")
	ErrVersionNotFound = New(http.StatusNotFound, "VERSION_NOT_FOUND", "Version not found")
	ErrEtagMismatch    = New(http.StatusConflict, "ETAG_MISMATCH", "Board was modified by another user")

	// Share errors
	ErrShareLinkNotFound = New(http.StatusNotFound, "SHARE_LINK_NOT_FOUND", "Share link not found")
	ErrShareLinkExpired  = New(http.StatusGone, "SHARE_LINK_EXPIRED", "Share link has expired")
)
'''

path = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'errors.go')
with open(path, 'w') as f:
    f.write(content.lstrip('\n'))
print(f"Wrote {os.path.getsize(path)} bytes to {path}")
