package persistence

import (
	"context"

	"github.com/d-sense/event-processor/pkg/models"
)

// Repository defines the interface for event persistence
type Repository interface {
	// Event operations
	SaveEvent(ctx context.Context, event *models.ProcessedEvent) error

	// Client configuration operations
	GetClientConfig(ctx context.Context, clientID string) (*models.ClientConfig, error)

	// Maintenance operations
	HealthCheck(ctx context.Context) error
}
