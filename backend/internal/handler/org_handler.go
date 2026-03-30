package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/darwinbatres/drawgo/backend/internal/middleware"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/response"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/validate"
	"github.com/darwinbatres/drawgo/backend/internal/service"
)

// OrgHandler handles organization HTTP endpoints.
type OrgHandler struct {
	orgs *service.OrgService
}

// NewOrgHandler creates an OrgHandler.
func NewOrgHandler(orgs *service.OrgService) *OrgHandler {
	return &OrgHandler{orgs: orgs}
}

type createOrgRequest struct {
	Name string `json:"name" validate:"required,min=1,max=100,trimmed"`
	Slug string `json:"slug" validate:"required,min=3,max=50,slug"`
}

type updateOrgRequest struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

// Create handles POST /api/v1/orgs.
func (h *OrgHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	var req createOrgRequest
	if apiErr := validate.DecodeAndValidate(r, &req); apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	org, apiErr := h.orgs.CreateOrg(r.Context(), userID, req.Name, req.Slug)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.Created(w, org)
}

// List handles GET /api/v1/orgs.
func (h *OrgHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	orgs, apiErr := h.orgs.ListOrgs(r.Context(), userID)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, orgs)
}

// Update handles PATCH /api/v1/orgs/{id}.
func (h *OrgHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	orgID := chi.URLParam(r, "id")
	if orgID == "" {
		response.Err(w, r, apierror.ErrBadRequest)
		return
	}

	var req updateOrgRequest
	if apiErr := validate.DecodeAndValidate(r, &req); apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	org, apiErr := h.orgs.UpdateOrg(r.Context(), userID, orgID, req.Name)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, org)
}

// Delete handles DELETE /api/v1/orgs/{id}.
func (h *OrgHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	orgID := chi.URLParam(r, "id")
	if orgID == "" {
		response.Err(w, r, apierror.ErrBadRequest)
		return
	}

	if apiErr := h.orgs.DeleteOrg(r.Context(), userID, orgID); apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.NoContent(w)
}
