package response

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
)

// envelope wraps all API responses in a consistent structure.
type envelope struct {
	Data  any            `json:"data,omitempty"`
	Error *apierror.Error `json:"error,omitempty"`
	Meta  *Meta          `json:"meta,omitempty"`
}

// Meta holds pagination and other metadata.
type Meta struct {
	Total  *int64 `json:"total,omitempty"`
	Limit  *int   `json:"limit,omitempty"`
	Offset *int   `json:"offset,omitempty"`
}

// JSON writes a JSON response with the given status code and data.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(envelope{Data: data})
	}
}

// JSONWithMeta writes a JSON response with pagination metadata.
func JSONWithMeta(w http.ResponseWriter, status int, data any, meta Meta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(envelope{Data: data, Meta: &meta})
}

// Err writes a structured error response. It logs internal errors with zerolog.
func Err(w http.ResponseWriter, r *http.Request, err *apierror.Error) {
	if err.Status >= 500 {
		log := zerolog.Ctx(r.Context())
		log.Error().Str("code", err.Code).Msg(err.Message)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(envelope{Error: err})
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Created writes a 201 Created response with the given data.
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, data)
}
