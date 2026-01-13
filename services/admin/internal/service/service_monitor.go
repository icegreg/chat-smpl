package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/icegreg/chat-smpl/services/admin/internal/model"
	"go.uber.org/zap"
)

// ServiceMonitor monitors microservices health
type ServiceMonitor struct {
	logger   *zap.Logger
	services map[string]serviceConfig
}

type serviceConfig struct {
	id          string
	name        string
	serviceType string
	healthURL   string
	metricsURL  string
}

// NewServiceMonitor creates a new service monitor
func NewServiceMonitor(logger *zap.Logger) *ServiceMonitor {
	// Define services to monitor
	services := map[string]serviceConfig{
		"voice-service": {
			id:          "voice-service",
			name:        "Voice Service",
			serviceType: "voice",
			healthURL:   "http://voice-service:8084/health",
			metricsURL:  "http://voice-service:8084/metrics",
		},
		"api-gateway": {
			id:          "api-gateway",
			name:        "API Gateway",
			serviceType: "gateway",
			healthURL:   "http://api-gateway:9180/health",
			metricsURL:  "http://api-gateway:9180/metrics",
		},
		"users-service": {
			id:          "users-service",
			name:        "Users Service",
			serviceType: "users",
			healthURL:   "http://users-service:8081/health",
			metricsURL:  "http://users-service:8081/metrics",
		},
		"files-service": {
			id:          "files-service",
			name:        "Files Service",
			serviceType: "files",
			healthURL:   "http://files-service:8082/health",
			metricsURL:  "http://files-service:8082/metrics",
		},
	}

	return &ServiceMonitor{
		logger:   logger,
		services: services,
	}
}

// ListServices returns list of all monitored services with their status
func (m *ServiceMonitor) ListServices(ctx context.Context) ([]model.Service, error) {
	var services []model.Service

	for _, cfg := range m.services {
		svc := model.Service{
			ID:   cfg.id,
			Name: cfg.name,
			Type: cfg.serviceType,
		}

		// Check health
		status, health := m.checkHealth(ctx, cfg.healthURL)
		svc.Status = status
		svc.Health = health

		now := time.Now()
		svc.LastCheck = &now

		services = append(services, svc)
	}

	return services, nil
}

// GetService returns status of a specific service
func (m *ServiceMonitor) GetService(ctx context.Context, serviceID string) (*model.Service, error) {
	cfg, ok := m.services[serviceID]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}

	svc := model.Service{
		ID:   cfg.id,
		Name: cfg.name,
		Type: cfg.serviceType,
	}

	// Check health
	status, health := m.checkHealth(ctx, cfg.healthURL)
	svc.Status = status
	svc.Health = health

	now := time.Now()
	svc.LastCheck = &now

	return &svc, nil
}

// checkHealth performs HTTP health check
func (m *ServiceMonitor) checkHealth(ctx context.Context, healthURL string) (model.ServiceStatus, string) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		m.logger.Error("failed to create health check request", zap.Error(err), zap.String("url", healthURL))
		return model.ServiceStatusUnknown, "failed to create request"
	}

	resp, err := client.Do(req)
	if err != nil {
		m.logger.Warn("health check failed", zap.Error(err), zap.String("url", healthURL))
		return model.ServiceStatusError, err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return model.ServiceStatusRunning, "healthy"
	}

	return model.ServiceStatusError, fmt.Sprintf("unhealthy: status %d", resp.StatusCode)
}
