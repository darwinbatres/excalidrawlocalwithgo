package handler

import (
	"crypto/subtle"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/response"
	"github.com/darwinbatres/drawgo/backend/internal/service"
)

// OAuthHandler handles OAuth2 authentication HTTP endpoints.
type OAuthHandler struct {
	oauth *service.OAuthService
}

// NewOAuthHandler creates an OAuthHandler.
func NewOAuthHandler(oauth *service.OAuthService) *OAuthHandler {
	return &OAuthHandler{oauth: oauth}
}

// Authorize handles GET /api/v1/auth/oauth/{provider}.
// Redirects the user to the OAuth provider's consent screen.
func (h *OAuthHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if !h.oauth.HasProvider(provider) {
		response.Err(w, r, apierror.New(http.StatusBadRequest, "INVALID_PROVIDER", "OAuth provider not configured"))
		return
	}

	authURL, state, err := h.oauth.GetAuthURL(provider)
	if err != nil {
		if apiErr, ok := err.(*apierror.Error); ok {
			response.Err(w, r, apiErr)
			return
		}
		response.Err(w, r, apierror.ErrInternal)
		return
	}

	// Store state in a short-lived, httpOnly cookie for CSRF validation on callback
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/api/v1/auth/oauth",
		HttpOnly: true,
		Secure:   h.oauth.IsProd(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300, // 5 minutes
	})

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// Callback handles GET /api/v1/auth/oauth/{provider}/callback.
// Exchanges the authorization code and signs the user in.
func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	// Validate state to prevent CSRF
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value == "" {
		response.Err(w, r, apierror.ErrOAuthStateMismatch)
		return
	}

	queryState := r.URL.Query().Get("state")
	if queryState == "" || subtle.ConstantTimeCompare([]byte(queryState), []byte(stateCookie.Value)) != 1 {
		response.Err(w, r, apierror.ErrOAuthStateMismatch)
		return
	}

	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/api/v1/auth/oauth",
		HttpOnly: true,
		Secure:   h.oauth.IsProd(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	// Check for provider error
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		response.Err(w, r, apierror.ErrOAuthProviderError.WithDetails(map[string]any{
			"provider_error": errMsg,
			"description":    r.URL.Query().Get("error_description"),
		}))
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		response.Err(w, r, apierror.ErrBadRequest.WithDetails(map[string]any{
			"code": "Authorization code is required",
		}))
		return
	}

	user, pair, callbackErr := h.oauth.HandleCallback(r.Context(), provider, code, r.UserAgent(), r.RemoteAddr)
	if callbackErr != nil {
		if apiErr, ok := callbackErr.(*apierror.Error); ok {
			response.Err(w, r, apiErr)
			return
		}
		response.Err(w, r, apierror.ErrInternal)
		return
	}

	service.SetAuthCookies(w, pair, h.oauth.IsProd())
	response.JSON(w, http.StatusOK, map[string]any{
		"user":   user.ToPublic(),
		"tokens": pair,
	})
}
