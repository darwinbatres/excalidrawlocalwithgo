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

// BackupHandler handles backup HTTP endpoints.
type BackupHandler struct {
	backups *service.BackupService
}

// NewBackupHandler creates a BackupHandler.
func NewBackupHandler(backups *service.BackupService) *BackupHandler {
	return &BackupHandler{backups: backups}
}

type createBackupRequest struct {
	Type string `json:"type" validate:"required,oneof=manual"`
}

type updateScheduleRequest struct {
	Enabled     bool   `json:"enabled"`
	CronExpr    string `json:"cronExpr" validate:"required,min=9,max=100"`
	KeepDaily   int    `json:"keepDaily" validate:"required,min=1,max=365"`
	KeepWeekly  int    `json:"keepWeekly" validate:"min=0,max=52"`
	KeepMonthly int    `json:"keepMonthly" validate:"min=0,max=120"`
}

// Create handles POST /api/v1/backups — trigger a manual backup.
func (h *BackupHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	var req createBackupRequest
	if apiErr := validate.DecodeAndValidate(r, &req); apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	meta, apiErr := h.backups.CreateBackup(r.Context(), req.Type, userID)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.Created(w, meta)
}

// List handles GET /api/v1/backups — list backups with pagination.
func (h *BackupHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	q := r.URL.Query()
	limit, offset := parsePagination(q.Get("limit"), q.Get("offset"))

	backups, total, apiErr := h.backups.ListBackups(r.Context(), limit, offset)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	total64 := int64(total)
	response.JSONWithMeta(w, http.StatusOK, backups, response.Meta{
		Total:  &total64,
		Limit:  &limit,
		Offset: &offset,
	})
}

// Get handles GET /api/v1/backups/{id} — get a single backup's metadata.
func (h *BackupHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Err(w, r, apierror.ErrBadRequest)
		return
	}

	meta, apiErr := h.backups.GetBackup(r.Context(), id)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, meta)
}

// Download handles GET /api/v1/backups/{id}/download — returns a presigned download URL.
func (h *BackupHandler) Download(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Err(w, r, apierror.ErrBadRequest)
		return
	}

	url, apiErr := h.backups.GetDownloadURL(r.Context(), id)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"url": url})
}

// Delete handles DELETE /api/v1/backups/{id} — delete a backup.
func (h *BackupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Err(w, r, apierror.ErrBadRequest)
		return
	}

	if apiErr := h.backups.DeleteBackup(r.Context(), id, userID); apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.NoContent(w)
}

// Restore handles POST /api/v1/backups/{id}/restore — restore from a backup.
func (h *BackupHandler) Restore(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Err(w, r, apierror.ErrBadRequest)
		return
	}

	if apiErr := h.backups.RestoreBackup(r.Context(), id, userID); apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "restored"})
}

// GetSchedule handles GET /api/v1/backups/schedule — get backup schedule.
func (h *BackupHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	sched, apiErr := h.backups.GetSchedule(r.Context())
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, sched)
}

// UpdateSchedule handles PUT /api/v1/backups/schedule — update backup schedule.
func (h *BackupHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	var req updateScheduleRequest
	if apiErr := validate.DecodeAndValidate(r, &req); apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	sched, apiErr := h.backups.UpdateSchedule(r.Context(), userID, req.Enabled, req.CronExpr, req.KeepDaily, req.KeepWeekly, req.KeepMonthly)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, sched)
}
