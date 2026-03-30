package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/darwinbatres/drawgo/backend/internal/middleware"
	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/response"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/validate"
	"github.com/darwinbatres/drawgo/backend/internal/service"
)

// ShareHandler handles share link HTTP endpoints.
type ShareHandler struct {
	shares *service.ShareService
}

// NewShareHandler creates a ShareHandler.
func NewShareHandler(shares *service.ShareService) *ShareHandler {
	return &ShareHandler{shares: shares}
}

type createShareLinkRequest struct {
	Role         string `json:"role" validate:"omitempty,oneof=VIEWER EDITOR"`
	ExpiresInSec *int64 `json:"expiresInSeconds,omitempty" validate:"omitempty,gt=0"`
}

// Create handles POST /api/v1/boards/{id}/share.
func (h *ShareHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	boardID := chi.URLParam(r, "id")
	if boardID == "" {
		response.Err(w, r, apierror.ErrBadRequest)
		return
	}

	var req createShareLinkRequest
	if apiErr := validate.DecodeAndValidate(r, &req); apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	role := models.BoardRole(req.Role)
	if role == "" {
		role = models.BoardRoleViewer
	}

	input := service.CreateShareLinkInput{
		BoardID: boardID,
		UserID:  userID,
		Role:    role,
	}

	if req.ExpiresInSec != nil && *req.ExpiresInSec > 0 {
		d := time.Duration(*req.ExpiresInSec) * time.Second
		input.ExpiresIn = &d
	}

	link, apiErr := h.shares.CreateShareLink(r.Context(), input)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusCreated, link)
}

// List handles GET /api/v1/boards/{id}/share.
func (h *ShareHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	boardID := chi.URLParam(r, "id")
	if boardID == "" {
		response.Err(w, r, apierror.ErrBadRequest)
		return
	}

	links, apiErr := h.shares.ListShareLinks(r.Context(), userID, boardID)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, links)
}

// Revoke handles DELETE /api/v1/boards/{id}/share/{linkId}.
func (h *ShareHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	linkID := chi.URLParam(r, "linkId")
	if linkID == "" {
		response.Err(w, r, apierror.ErrBadRequest)
		return
	}

	apiErr := h.shares.RevokeShareLink(r.Context(), userID, linkID)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// GetShared handles GET /api/v1/share/{token} — public, no auth required.
func (h *ShareHandler) GetShared(w http.ResponseWriter, r *http.Request) {
	tok := chi.URLParam(r, "token")
	if tok == "" {
		response.Err(w, r, apierror.ErrBadRequest)
		return
	}

	data, apiErr := h.shares.GetSharedBoard(r.Context(), tok)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, data)
}
