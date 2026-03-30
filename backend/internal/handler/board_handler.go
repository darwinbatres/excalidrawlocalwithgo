package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/darwinbatres/drawgo/backend/internal/middleware"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/response"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/validate"
	"github.com/darwinbatres/drawgo/backend/internal/repository"
	"github.com/darwinbatres/drawgo/backend/internal/service"
)

// BoardHandler handles board HTTP endpoints.
type BoardHandler struct {
	boards *service.BoardService
}

// NewBoardHandler creates a BoardHandler.
func NewBoardHandler(boards *service.BoardService) *BoardHandler {
	return &BoardHandler{boards: boards}
}

type createBoardRequest struct {
	Title        string   `json:"title" validate:"required,min=1,max=200"`
	Description  *string  `json:"description,omitempty" validate:"omitempty,max=2000"`
	Tags         []string `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1,max=50"`
	SceneJSON    any      `json:"sceneJson,omitempty"`
	AppStateJSON any      `json:"appStateJson,omitempty"`
}

type updateBoardRequest struct {
	Title       *string  `json:"title,omitempty" validate:"omitempty,min=1,max=200"`
	Description *string  `json:"description,omitempty" validate:"omitempty,max=2000"`
	Tags        []string `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1,max=50"`
	IsArchived  *bool    `json:"isArchived,omitempty"`
}

// Create handles POST /api/v1/orgs/{id}/boards.
func (h *BoardHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	var req createBoardRequest
	if apiErr := validate.DecodeAndValidate(r, &req); apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	sceneJSON, err := marshalToRaw(req.SceneJSON)
	if err != nil {
		response.Err(w, r, apierror.ErrBadRequest.WithMessage("Invalid sceneJson"))
		return
	}
	if sceneJSON == nil {
		sceneJSON = json.RawMessage(`{"elements":[],"files":{}}`)
	}

	appStateJSON, _ := marshalToRaw(req.AppStateJSON)

	board, apiErr := h.boards.CreateBoard(r.Context(), userID, service.CreateBoardInput{
		OrgID:        orgID,
		Title:        req.Title,
		Description:  req.Description,
		Tags:         req.Tags,
		SceneJSON:    sceneJSON,
		AppStateJSON: appStateJSON,
	})
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.Created(w, board)
}

// List handles GET /api/v1/orgs/{id}/boards.
func (h *BoardHandler) List(w http.ResponseWriter, r *http.Request) {
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

	var isArchived *bool
	if v := q.Get("archived"); v != "" {
		b := v == "true"
		isArchived = &b
	}

	var tags []string
	if t := q.Get("tags"); t != "" {
		for _, tag := range strings.Split(t, ",") {
			if trimmed := strings.TrimSpace(tag); trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
	}

	params := repository.BoardSearchParams{
		OrgID:      orgID,
		Query:      q.Get("q"),
		Tags:       tags,
		IsArchived: isArchived,
		Limit:      limit,
		Offset:     offset,
	}

	result, apiErr := h.boards.SearchBoards(r.Context(), userID, params)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSONWithMeta(w, http.StatusOK, result.Boards, response.Meta{
		Total:  &result.Total,
		Limit:  &limit,
		Offset: &offset,
	})
}

// Get handles GET /api/v1/boards/{id}.
func (h *BoardHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	bv, apiErr := h.boards.GetBoard(r.Context(), userID, boardID)
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, bv)
}

// Update handles PATCH /api/v1/boards/{id}.
func (h *BoardHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	var req updateBoardRequest
	if apiErr := validate.DecodeAndValidate(r, &req); apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	board, apiErr := h.boards.UpdateBoard(r.Context(), userID, boardID, service.UpdateBoardInput{
		Title:       req.Title,
		Description: req.Description,
		Tags:        req.Tags,
		IsArchived:  req.IsArchived,
	})
	if apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.JSON(w, http.StatusOK, board)
}

// Delete handles DELETE /api/v1/boards/{id}.
func (h *BoardHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	if apiErr := h.boards.DeleteBoard(r.Context(), userID, boardID); apiErr != nil {
		response.Err(w, r, apiErr)
		return
	}

	response.NoContent(w)
}
