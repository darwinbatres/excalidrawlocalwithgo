package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/darwinbatres/drawgo/backend/internal/middleware"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/response"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/validate"
	"github.com/darwinbatres/drawgo/backend/internal/service"
)

// VersionHandler handles board version HTTP endpoints.
type VersionHandler struct {
	boards *service.BoardService
}

// NewVersionHandler creates a VersionHandler.
func NewVersionHandler(boards *service.BoardService) *VersionHandler {
	return &VersionHandler{boards: boards}
}

type saveVersionRequest struct {
	SceneJSON    any     `json:"sceneJson" validate:"required"`
	AppStateJSON any     `json:"appStateJson,omitempty"`
	Label        *string `json:"label,omitempty" validate:"omitempty,max=255"`
	ExpectedEtag *string `json:"expectedEtag,omitempty"`
	Thumbnail    *string `json:"thumbnail,omitempty"`
}

// Save handles POST /api/v1/boards/{id}/versions.
func (h *VersionHandler) Save(w http.ResponseWriter, r *http.Request) {
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

	var req saveVersionRequest
	if apiErr := validate.DecodeAndValidate(r, &req); apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	sceneJSON, err := marshalToRaw(req.SceneJSON)
	if err != nil {
		response.Err(w, r, apierror.ErrBadRequest.WithMessage("Invalid sceneJson"))
		return
	}

	appStateJSON, _ := marshalToRaw(req.AppStateJSON)

	result, apiErr := h.boards.SaveVersion(r.Context(), service.SaveVersionInput{
		BoardID:      boardID,
		UserID:       userID,
		SceneJSON:    sceneJSON,
		AppStateJSON: appStateJSON,
		Label:        req.Label,
		ExpectedEtag: req.ExpectedEtag,
		Thumbnail:    req.Thumbnail,
	})
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	if result.Conflict {
		response.Err(w, r, apierror.ErrEtagMismatch.WithDetails(map[string]any{
			"currentEtag": result.CurrentEtag,
		}))
		return
	}

	response.Created(w, result)
}

// List handles GET /api/v1/boards/{id}/versions.
func (h *VersionHandler) List(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()
	limit, offset := parsePagination(q.Get("limit"), q.Get("offset"))

	versions, total, apiErr := h.boards.ListVersions(r.Context(), userID, boardID, limit, offset)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSONWithMeta(w, http.StatusOK, versions, response.Meta{
		Total:  &total,
		Limit:  &limit,
		Offset: &offset,
	})
}

// Get handles GET /api/v1/boards/{id}/versions/{version}.
func (h *VersionHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	boardID := chi.URLParam(r, "id")
	versionStr := chi.URLParam(r, "version")
	if boardID == "" || versionStr == "" {
		response.Err(w, r, apierror.ErrBadRequest)
		return
	}

	versionNum, err := strconv.Atoi(versionStr)
	if err != nil || versionNum < 1 {
		response.Err(w, r, apierror.ErrBadRequest.WithMessage("Version must be a positive integer"))
		return
	}

	version, apiErr := h.boards.GetVersion(r.Context(), userID, boardID, versionNum)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, version)
}

// Restore handles POST /api/v1/boards/{id}/versions/{version}/restore.
func (h *VersionHandler) Restore(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	boardID := chi.URLParam(r, "id")
	versionStr := chi.URLParam(r, "version")
	if boardID == "" || versionStr == "" {
		response.Err(w, r, apierror.ErrBadRequest)
		return
	}

	versionNum, err := strconv.Atoi(versionStr)
	if err != nil || versionNum < 1 {
		response.Err(w, r, apierror.ErrBadRequest.WithMessage("Version must be a positive integer"))
		return
	}

	result, apiErr := h.boards.RestoreVersion(r.Context(), userID, boardID, versionNum)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.Created(w, result)
}
