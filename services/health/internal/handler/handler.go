package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/icegreg/chat-smpl/services/health/internal/checker"
)

type Handler struct {
	checker *checker.Checker
	log     *slog.Logger
}

func NewHandler(c *checker.Checker, log *slog.Logger) *Handler {
	return &Handler{
		checker: c,
		log:     log,
	}
}

// Health is a simple liveness check
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// SystemHealth returns detailed system health status with metrics.
// Always returns HTTP 200 - use status field in response body for actual health state.
// For HTTP status codes based on health, use SystemHealthCheck endpoint.
func (h *Handler) SystemHealth(w http.ResponseWriter, r *http.Request) {
	result := h.checker.GetLastResult()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.log.Error("failed to encode health response", "error", err)
	}
}

// SystemHealthCheck returns system health with HTTP status codes reflecting health state.
// Use this endpoint for load balancers, k8s probes, or monitoring systems.
// Returns: 200 (ok/degraded), 503 (down), 500 (unknown)
func (h *Handler) SystemHealthCheck(w http.ResponseWriter, r *http.Request) {
	result := h.checker.GetLastResult()

	w.Header().Set("Content-Type", "application/json")

	// Set HTTP status based on health status
	switch result.Status {
	case checker.StatusOK, checker.StatusDegraded:
		w.WriteHeader(http.StatusOK)
	case checker.StatusDown:
		w.WriteHeader(http.StatusServiceUnavailable)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.log.Error("failed to encode health response", "error", err)
	}
}
