package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/darwinbatres/drawgo/backend/internal/middleware"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/response"
	"github.com/darwinbatres/drawgo/backend/internal/repository"
	"github.com/darwinbatres/drawgo/backend/internal/service"
)

// AuditHandler handles audit log and stats HTTP endpoints.
type AuditHandler struct {
	audits *service.AuditService
}

// NewAuditHandler creates an AuditHandler.
func NewAuditHandler(audits *service.AuditService) *AuditHandler {
	return &AuditHandler{audits: audits}
}

// List handles GET /api/v1/orgs/{id}/audit.
// Query params: actorId, action, targetType, targetId, startDate, endDate, limit, offset.
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()
	limit, offset := parsePagination(q.Get("limit"), q.Get("offset"))

	params := repository.AuditQueryParams{
		ActorID:    q.Get("actorId"),
		Action:     q.Get("action"),
		TargetType: q.Get("targetType"),
		TargetID:   q.Get("targetId"),
		Limit:      limit,
		Offset:     offset,
	}

	if sd := q.Get("startDate"); sd != "" {
		if t, err := time.Parse(time.RFC3339, sd); err == nil {
			params.StartDate = &t
		}
	}
	if ed := q.Get("endDate"); ed != "" {
		if t, err := time.Parse(time.RFC3339, ed); err == nil {
			params.EndDate = &t
		}
	}

	result, apiErr := h.audits.ListAuditLogs(r.Context(), userID, orgID, params)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSONWithMeta(w, http.StatusOK, result.Events, response.Meta{
		Total:  &result.Total,
		Limit:  &limit,
		Offset: &offset,
	})
}

// Stats handles GET /api/v1/orgs/{id}/audit/stats.
// Query params: days (default 30).
func (h *AuditHandler) Stats(w http.ResponseWriter, r *http.Request) {
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

	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 {
			days = v
		}
	}
	if days > 365 {
		days = 365
	}

	stats, apiErr := h.audits.GetAuditStats(r.Context(), userID, orgID, days)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, stats)
}

// SystemStats handles GET /api/v1/stats.
// Returns system-wide counts for all major tables.
func (h *AuditHandler) SystemStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	counts, apiErr := h.audits.SystemStats(r.Context())
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, counts)
}

// OrgStats handles GET /api/v1/orgs/{id}/stats.
// Returns org-scoped counts and CRUD breakdown. Requires VIEWER+.
func (h *AuditHandler) OrgStats(w http.ResponseWriter, r *http.Request) {
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

	stats, apiErr := h.audits.OrgStats(r.Context(), userID, orgID)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, stats)
}
