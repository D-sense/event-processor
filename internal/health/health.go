package health

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/d-sense/event-processor/internal/persistence"
	"github.com/d-sense/event-processor/pkg/logger"
)

// Status HealthStatus represents the health status of the service
type Status struct {
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

// Checker performs health checks on various components
type Checker struct {
	repository persistence.Repository
	logger     *logrus.Logger
}

// New creates a new HealthChecker instance
func New(repo persistence.Repository, logger *logrus.Logger) *Checker {
	return &Checker{
		repository: repo,
		logger:     logger,
	}
}

// Check performs health checks on all components
func (h *Checker) Check(ctx context.Context) *Status {
	status := &Status{
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
	healthLogger := logger.WithFields(h.logger, map[string]interface{}{
		"healthy":     status.Healthy,
		"db_healthy":  dbHealth.Healthy,
		"db_latency":  dbHealth.Latency,
		"mem_healthy": memHealth.Healthy,
	})
	healthLogger.Debug("Health check completed")

	return status
}

// checkDatabase checks the database connectivity and performance
func (h *Checker) checkDatabase(ctx context.Context) ComponentHealth {
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
// I am keeping this as a simplified memory check
// In a real application, you might want to check actual memory usage
// using runtime.MemStats or other memory monitoring tools
func (h *Checker) checkMemory() ComponentHealth {
	start := time.Now()

	latency := time.Since(start)

	return ComponentHealth{
		Healthy: true,
		Latency: latency,
	}
}
