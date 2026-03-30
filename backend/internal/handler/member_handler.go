package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/darwinbatres/drawgo/backend/internal/middleware"
	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/response"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/validate"
	"github.com/darwinbatres/drawgo/backend/internal/service"
)

// MemberHandler handles membership HTTP endpoints.
type MemberHandler struct {
	orgs *service.OrgService
}

// NewMemberHandler creates a MemberHandler.
func NewMemberHandler(orgs *service.OrgService) *MemberHandler {
	return &MemberHandler{orgs: orgs}
}

type inviteMemberRequest struct {
	Email string        `json:"email" validate:"required,email"`
	Role  models.OrgRole `json:"role" validate:"omitempty,oneof=ADMIN MEMBER VIEWER"`
}

type updateRoleRequest struct {
	Role models.OrgRole `json:"role" validate:"required,oneof=ADMIN MEMBER VIEWER"`
}

// List handles GET /api/v1/orgs/{id}/members.
func (h *MemberHandler) List(w http.ResponseWriter, r *http.Request) {
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

	members, apiErr := h.orgs.ListMembers(r.Context(), userID, orgID)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, members)
}

// Invite handles POST /api/v1/orgs/{id}/members.
func (h *MemberHandler) Invite(w http.ResponseWriter, r *http.Request) {
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

	var req inviteMemberRequest
	if apiErr := validate.DecodeAndValidate(r, &req); apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	// Default role to MEMBER if not specified
	role := req.Role
	if role == "" {
		role = models.OrgRoleMember
	}

	member, apiErr := h.orgs.InviteMember(r.Context(), userID, orgID, req.Email, role)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.Created(w, member)
}

// UpdateRole handles PATCH /api/v1/orgs/{id}/members/{membershipId}.
func (h *MemberHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	orgID := chi.URLParam(r, "id")
	membershipID := chi.URLParam(r, "membershipId")
	if orgID == "" || membershipID == "" {
		response.Err(w, r, apierror.ErrBadRequest)
		return
	}

	var req updateRoleRequest
	if apiErr := validate.DecodeAndValidate(r, &req); apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	membership, apiErr := h.orgs.UpdateMemberRole(r.Context(), userID, orgID, membershipID, req.Role)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, membership)
}

// Remove handles DELETE /api/v1/orgs/{id}/members/{membershipId}.
func (h *MemberHandler) Remove(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	orgID := chi.URLParam(r, "id")
	membershipID := chi.URLParam(r, "membershipId")
	if orgID == "" || membershipID == "" {
		response.Err(w, r, apierror.ErrBadRequest)
		return
	}

	if apiErr := h.orgs.RemoveMember(r.Context(), userID, orgID, membershipID); apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.NoContent(w)
}
