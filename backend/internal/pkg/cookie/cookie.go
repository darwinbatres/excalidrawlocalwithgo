package cookie

import "net/http"

// Base names for auth cookies.
const (
	baseAccessName  = "access_token"
	baseRefreshName = "refresh_token"
)

// AccessName returns the cookie name for the access token.
// In production (secure=true), uses the __Host- prefix which
// requires Secure, Path=/, and no Domain — preventing cookie
// tossing and subdomain attacks per RFC 6265bis.
func AccessName(secure bool) string {
	if secure {
		return "__Host-" + baseAccessName
	}
	return baseAccessName
}

// RefreshName returns the cookie name for the refresh token.
// In production (secure=true), uses the __Secure- prefix which
// requires the Secure flag. We use __Secure- instead of __Host-
// because the refresh cookie has a scoped Path (/api/v1/auth).
func RefreshName(secure bool) string {
	if secure {
		return "__Secure-" + baseRefreshName
	}
	return baseRefreshName
}

// SetAccess writes the access token cookie with hardened attributes.
func SetAccess(w http.ResponseWriter, value string, secure bool, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     AccessName(secure),
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})
}

// SetRefresh writes the refresh token cookie with hardened attributes.
func SetRefresh(w http.ResponseWriter, value string, secure bool, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshName(secure),
		Value:    value,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
	})
}

// ClearAccess removes the access token cookie.
func ClearAccess(w http.ResponseWriter, secure bool) {
	SetAccess(w, "", secure, -1)
}

// ClearRefresh removes the refresh token cookie.
func ClearRefresh(w http.ResponseWriter, secure bool) {
	SetRefresh(w, "", secure, -1)
}

// ReadAccess extracts the access token from the request cookie,
// trying the __Host- prefixed name first (production), then unprefixed (dev).
func ReadAccess(r *http.Request) string {
	if c, err := r.Cookie("__Host-" + baseAccessName); err == nil && c.Value != "" {
		return c.Value
	}
	if c, err := r.Cookie(baseAccessName); err == nil && c.Value != "" {
		return c.Value
	}
	return ""
}

// ReadRefresh extracts the refresh token from the request cookie,
// trying the __Secure- prefixed name first (production), then unprefixed (dev).
func ReadRefresh(r *http.Request) string {
	if c, err := r.Cookie("__Secure-" + baseRefreshName); err == nil && c.Value != "" {
		return c.Value
	}
	if c, err := r.Cookie(baseRefreshName); err == nil && c.Value != "" {
		return c.Value
	}
	return ""
}
