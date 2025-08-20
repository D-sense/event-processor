package persistence

import (
	"context"

	"github.com/d-sense/event-processor/pkg/models"
)

// Repository defines the interface for event persistence
type Repository interface {
	// Event operations
	SaveEvent(ctx context.Context, event *models.ProcessedEvent) error
	GetEvent(ctx context.Context, eventID string) (*models.ProcessedEvent, error)
	GetEventsByClient(ctx context.Context, clientID string, limit int) ([]*models.ProcessedEvent, error)
	GetEventsByStatus(ctx context.Context, status models.EventStatus, limit int) ([]*models.ProcessedEvent, error)
	UpdateEventStatus(ctx context.Context, eventID string, status models.EventStatus, errorMsg string) error

	// Client configuration operations
	SaveClientConfig(ctx context.Context, config *models.ClientConfig) error
	GetClientConfig(ctx context.Context, clientID string) (*models.ClientConfig, error)

	// Maintenance operations
	DeleteExpiredEvents(ctx context.Context) error
	HealthCheck(ctx context.Context) error
	GetEventStats(ctx context.Context, clientID string, hours int) (*models.EventMetrics, error)
}
