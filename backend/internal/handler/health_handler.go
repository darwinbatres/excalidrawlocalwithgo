package handler

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/buildinfo"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/response"
)

// Pinger checks connectivity to a dependency.
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler provides liveness, readiness, and version endpoints.
type HealthHandler struct {
	pool      *pgxpool.Pool
	storage   Pinger
	startTime time.Time
}

// NewHealthHandler creates a new HealthHandler. storage may be nil.
func NewHealthHandler(pool *pgxpool.Pool, storage Pinger) *HealthHandler {
	return &HealthHandler{pool: pool, storage: storage, startTime: time.Now()}
}

// Health returns basic liveness info (always 200 if the process is running).
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	response.JSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"uptime":      time.Since(h.startTime).String(),
		"goroutines":  runtime.NumGoroutine(),
		"memoryAlloc": m.Alloc,
		"goVersion":   runtime.Version(),
		"version":     buildinfo.Version,
		"commit":      buildinfo.CommitSHA,
		"buildTime":   buildinfo.BuildTime,
	})
}

// Ready checks that all dependencies (DB, S3) are reachable.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	deps := make(map[string]string)
	healthy := true

	// Database check
	if err := h.checkDep(ctx, h.pool); err != nil {
		deps["database"] = "unavailable"
		healthy = false
	} else {
		deps["database"] = "connected"
	}

	// S3 storage check
	if h.storage != nil {
		if err := h.checkDep(ctx, h.storage); err != nil {
			deps["storage"] = "unavailable"
			healthy = false
		} else {
			deps["storage"] = "connected"
		}
	}

	if !healthy {
		response.Err(w, r, apierror.ErrServiceUnavailable.WithDetails(map[string]any{
			"dependencies": deps,
		}))
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"status":       "ready",
		"dependencies": deps,
	})
}

// Version returns build version information.
func (h *HealthHandler) Version(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]any{
		"version":   buildinfo.Version,
		"commit":    buildinfo.CommitSHA,
		"buildTime": buildinfo.BuildTime,
		"goVersion": runtime.Version(),
	})
}

// checkDep checks a dependency with a bounded timeout.
func (h *HealthHandler) checkDep(ctx context.Context, p Pinger) error {
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return p.Ping(checkCtx)
}
