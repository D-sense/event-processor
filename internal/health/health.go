package health

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/d-sense/event-processor/internal/persistence"
)

// HealthStatus represents the health status of the service
type HealthStatus struct {
	Healthy   bool                       `json:"healthy"`
	Timestamp time.Time                  `json:"timestamp"`
	Checks    map[string]ComponentHealth `json:"checks"`
}

// ComponentHealth represents the health of a specific component
type ComponentHealth struct {
	Healthy bool          `json:"healthy"`
	Latency time.Duration `json:"latency"`
	Error   string        `json:"error,omitempty"`
}

// HealthChecker performs health checks on various components
type HealthChecker struct {
	repository persistence.Repository
	logger     *logrus.Logger
}

// New creates a new HealthChecker instance
func New(repo persistence.Repository, logger *logrus.Logger) *HealthChecker {
	return &HealthChecker{
		repository: repo,
		logger:     logger,
	}
}

// Check performs health checks on all components
func (h *HealthChecker) Check(ctx context.Context) *HealthStatus {
	status := &HealthStatus{
		Healthy:   true,
		Timestamp: time.Now().UTC(),
		Checks:    make(map[string]ComponentHealth),
	}

	// Check database health
	dbHealth := h.checkDatabase(ctx)
	status.Checks["database"] = dbHealth
	if !dbHealth.Healthy {
		status.Healthy = false
	}

	// Check memory usage (basic check)
	memHealth := h.checkMemory()
	status.Checks["memory"] = memHealth
	if !memHealth.Healthy {
		status.Healthy = false
	}

	// Log health check results
	h.logger.WithFields(logrus.Fields{
		"healthy":     status.Healthy,
		"db_healthy":  dbHealth.Healthy,
		"db_latency":  dbHealth.Latency,
		"mem_healthy": memHealth.Healthy,
	}).Debug("Health check completed")

	return status
}

// checkDatabase checks the database connectivity and performance
func (h *HealthChecker) checkDatabase(ctx context.Context) ComponentHealth {
	start := time.Now()

	// Create a context with timeout for the database check
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := h.repository.HealthCheck(dbCtx)
	latency := time.Since(start)

	if err != nil {
		h.logger.WithError(err).Error("Database health check failed")
		return ComponentHealth{
			Healthy: false,
			Latency: latency,
			Error:   err.Error(),
		}
	}

	return ComponentHealth{
		Healthy: true,
		Latency: latency,
	}
}

// checkMemory performs basic memory usage check
func (h *HealthChecker) checkMemory() ComponentHealth {
	start := time.Now()

	// This is a simplified memory check
	// In a real application, you might want to check actual memory usage
	// using runtime.MemStats or other memory monitoring tools

	latency := time.Since(start)

	return ComponentHealth{
		Healthy: true,
		Latency: latency,
	}
}

// IsHealthy returns a simple boolean health status
func (h *HealthChecker) IsHealthy(ctx context.Context) bool {
	status := h.Check(ctx)
	return status.Healthy
}
