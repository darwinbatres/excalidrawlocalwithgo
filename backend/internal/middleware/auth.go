package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/cookie"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/jwt"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/response"
)

type contextKey string

const (
	UserIDKey contextKey = "userID"
	EmailKey  contextKey = "email"
)

// Auth creates a middleware that validates JWT access tokens from cookies or Authorization header.
func Auth(jwtManager *jwt.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractToken(r)
			if tokenStr == "" {
				response.Err(w, r, apierror.ErrUnauthorized)
				return
			}

			claims, err := jwtManager.ValidateAccessToken(tokenStr)
			if err != nil {
				response.Err(w, r, apierror.ErrTokenInvalid)
				return
			}

			// Inject user info into request context
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, EmailKey, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromCtx extracts the authenticated user ID from the request context.
func UserIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

// EmailFromCtx extracts the authenticated user email from the request context.
func EmailFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(EmailKey).(string); ok {
		return v
	}
	return ""
}

// extractToken gets the JWT from the cookie or Authorization header.
func extractToken(r *http.Request) string {
	// Try httpOnly cookie first (supports __Host- prefixed names)
	if tok := cookie.ReadAccess(r); tok != "" {
		return tok
	}

	// Fall back to Authorization: Bearer <token>
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	return ""
}
