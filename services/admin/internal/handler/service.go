package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/icegreg/chat-smpl/services/admin/internal/model"
	"github.com/icegreg/chat-smpl/services/admin/internal/service"
	"go.uber.org/zap"
)

// ServiceHandler handles service monitoring HTTP endpoints
type ServiceHandler struct {
	monitor *service.ServiceMonitor
	logger  *zap.Logger
}

// NewServiceHandler creates a new service handler
func NewServiceHandler(monitor *service.ServiceMonitor, logger *zap.Logger) *ServiceHandler {
	return &ServiceHandler{
		monitor: monitor,
		logger:  logger,
	}
}

// ListServices handles GET /api/services
func (h *ServiceHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	services, err := h.monitor.ListServices(ctx)
	if err != nil {
		h.logger.Error("failed to list services", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "failed to list services", err)
		return
	}

	resp := model.ServicesResponse{
		Services: services,
		Total:    len(services),
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// GetService handles GET /api/services/{id}
func (h *ServiceHandler) GetService(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	serviceID := chi.URLParam(r, "id")

	svc, err := h.monitor.GetService(ctx, serviceID)
	if err != nil {
		h.logger.Error("failed to get service", zap.Error(err), zap.String("service_id", serviceID))
		h.writeError(w, http.StatusNotFound, "service not found", err)
		return
	}

	h.writeJSON(w, http.StatusOK, svc)
}

// writeJSON writes JSON response
func (h *ServiceHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON response", zap.Error(err))
	}
}

// writeError writes error response
func (h *ServiceHandler) writeError(w http.ResponseWriter, status int, message string, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	resp := model.ErrorResponse{
		Error:   message,
		Message: errMsg,
	}

	h.writeJSON(w, status, resp)
}
