package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/d-sense/event-processor/internal/persistence"
	"github.com/d-sense/event-processor/internal/validator"
	"github.com/d-sense/event-processor/pkg/models"
)

// EventProcessor handles the core event processing logic
type EventProcessor struct {
	repository persistence.Repository
	validator  *validator.Validator
	logger     *logrus.Logger
}

// New creates a new EventProcessor instance
func New(repo persistence.Repository, validator *validator.Validator, logger *logrus.Logger) *EventProcessor {
	return &EventProcessor{
		repository: repo,
		validator:  validator,
		logger:     logger,
	}
}

// ProcessEvent processes an incoming event
func (p *EventProcessor) ProcessEvent(ctx context.Context, eventData interface{}) error {
	startTime := time.Now()

	// Generate correlation ID for tracing
	correlationID := fmt.Sprintf("proc_%d", time.Now().UnixNano())

	logger := p.logger.WithFields(logrus.Fields{
		"correlation_id": correlationID,
		"component":      "event_processor",
	})

	logger.Debug("Starting event processing")

	// Step 1: Validate and parse the event
	event, err := p.validator.ValidateAndParseEvent(eventData)
	if err != nil {
		logger.WithError(err).Error("Event validation failed")
		return fmt.Errorf("validation failed: %w", err)
	}

	logger = logger.WithFields(logrus.Fields{
		"event_id":   event.EventID,
		"event_type": event.EventType,
		"client_id":  event.ClientID,
	})

	logger.Info("Event validated successfully")

	// Step 2: Perform event triage
	processedEvent, err := p.triageEvent(ctx, event, logger)
	if err != nil {
		logger.WithError(err).Error("Event triage failed")
		return fmt.Errorf("triage failed: %w", err)
	}

	// Step 3: Persist the event
	if err := p.repository.SaveEvent(ctx, processedEvent); err != nil {
		logger.WithError(err).Error("Failed to persist event")
		return fmt.Errorf("persistence failed: %w", err)
	}

	processingTime := time.Since(startTime)
	logger.WithFields(logrus.Fields{
		"processing_time_ms": processingTime.Milliseconds(),
		"status":             processedEvent.Status,
	}).Info("Event processed successfully")

	return nil
}

// triageEvent performs event triage and routing logic
func (p *EventProcessor) triageEvent(ctx context.Context, event *models.Event, logger *logrus.Entry) (*models.ProcessedEvent, error) {
	processedEvent := event.ToProcessedEvent()

	// Perform event-type specific processing
	switch event.EventType {
	case models.EventTypeMonitoring:
		if err := p.processMonitoringEvent(event, processedEvent, logger); err != nil {
			return nil, err
		}
	case models.EventTypeUserAction:
		if err := p.processUserActionEvent(event, processedEvent, logger); err != nil {
			return nil, err
		}
	case models.EventTypeTransaction:
		if err := p.processTransactionEvent(event, processedEvent, logger); err != nil {
			return nil, err
		}
	case models.EventTypeIntegration:
		if err := p.processIntegrationEvent(event, processedEvent, logger); err != nil {
			return nil, err
		}
	default:
		logger.WithField("event_type", event.EventType).Warn("Unknown event type")
		processedEvent.Status = models.EventStatusFailed
		processedEvent.ErrorMsg = fmt.Sprintf("unknown event type: %s", event.EventType)
	}

	// Validate client permissions
	if err := p.validateClientPermissions(ctx, event.ClientID, event.EventType, logger); err != nil {
		logger.WithError(err).Warn("Client permission validation failed")
		processedEvent.Status = models.EventStatusFailed
		processedEvent.ErrorMsg = fmt.Sprintf("client permission error: %v", err)
	}

	// Set final status if not already set
	if processedEvent.Status == models.EventStatusPending {
		processedEvent.Status = models.EventStatusProcessed
	}

	return processedEvent, nil
}

// processMonitoringEvent handles monitoring-specific logic
func (p *EventProcessor) processMonitoringEvent(event *models.Event, processedEvent *models.ProcessedEvent, logger *logrus.Entry) error {
	logger.Debug("Processing monitoring event")

	// Extract monitoring-specific fields
	if severity, ok := event.Payload["severity"]; ok {
		if severityStr, ok := severity.(string); ok {
			// Route high-severity events differently
			if severityStr == "critical" || severityStr == "high" {
				logger.WithField("severity", severityStr).Info("High-severity monitoring event detected")
				// Add priority flag to payload for downstream processing
				processedEvent.Payload = make(map[string]interface{})
				for k, v := range event.Payload {
					processedEvent.Payload[k] = v
				}
				processedEvent.Payload["priority"] = "high"
			}
		}
	}

	return nil
}

// processUserActionEvent handles user action-specific logic
func (p *EventProcessor) processUserActionEvent(event *models.Event, processedEvent *models.ProcessedEvent, logger *logrus.Entry) error {
	logger.Debug("Processing user action event")

	// Validate user action payload
	requiredFields := []string{"userId", "action", "resource"}
	for _, field := range requiredFields {
		if _, exists := event.Payload[field]; !exists {
			return fmt.Errorf("missing required field for user action: %s", field)
		}
	}

	// Add processing timestamp for audit trail
	if processedEvent.Payload == nil {
		processedEvent.Payload = make(map[string]interface{})
	}
	for k, v := range event.Payload {
		processedEvent.Payload[k] = v
	}
	processedEvent.Payload["processedAt"] = time.Now().UTC().Format(time.RFC3339)

	return nil
}

// processTransactionEvent handles transaction-specific logic
func (p *EventProcessor) processTransactionEvent(event *models.Event, processedEvent *models.ProcessedEvent, logger *logrus.Entry) error {
	logger.Debug("Processing transaction event")

	// Validate transaction payload
	requiredFields := []string{"transactionId", "amount", "currency"}
	for _, field := range requiredFields {
		if _, exists := event.Payload[field]; !exists {
			return fmt.Errorf("missing required field for transaction: %s", field)
		}
	}

	// Add transaction processing metadata
	if processedEvent.Payload == nil {
		processedEvent.Payload = make(map[string]interface{})
	}
	for k, v := range event.Payload {
		processedEvent.Payload[k] = v
	}

	// Check for high-value transactions
	if amountFloat, ok := event.Payload["amount"].(float64); ok {
		if amountFloat > 10000 { // Threshold for high-value transactions
			processedEvent.Payload["highValue"] = true
			logger.WithField("amount", amountFloat).Info("High-value transaction detected")
		}
	}

	return nil
}

// processIntegrationEvent handles integration-specific logic
func (p *EventProcessor) processIntegrationEvent(event *models.Event, processedEvent *models.ProcessedEvent, logger *logrus.Entry) error {
	logger.Debug("Processing integration event")

	// Validate integration payload
	requiredFields := []string{"source", "target", "operation"}
	for _, field := range requiredFields {
		if _, exists := event.Payload[field]; !exists {
			return fmt.Errorf("missing required field for integration: %s", field)
		}
	}

	// Add integration metadata
	if processedEvent.Payload == nil {
		processedEvent.Payload = make(map[string]interface{})
	}
	for k, v := range event.Payload {
		processedEvent.Payload[k] = v
	}
	processedEvent.Payload["integrationProcessedAt"] = time.Now().UTC().Format(time.RFC3339)

	return nil
}

// validateClientPermissions validates if client has permission to send this event type
func (p *EventProcessor) validateClientPermissions(ctx context.Context, clientID string, eventType models.EventType, logger *logrus.Entry) error {
	// Check if client exists and is active
	clientConfig, err := p.repository.GetClientConfig(ctx, clientID)
	if err != nil {
		// If client config doesn't exist, allow by default (could be configurable)
		logger.WithField("client_id", clientID).Debug("Client config not found, allowing by default")
		return nil
	}

	if !clientConfig.Active {
		return fmt.Errorf("client %s is not active", clientID)
	}

	// Check if client is allowed to send this event type
	allowed := false
	for _, allowedType := range clientConfig.AllowedTypes {
		if allowedType == eventType {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("client %s is not allowed to send events of type %s", clientID, eventType)
	}

	return nil
}

// GetMetrics returns processing metrics
func (p *EventProcessor) GetMetrics() (*models.EventMetrics, error) {
	// This would typically aggregate metrics from the repository
	// For now, we return basic metrics
	return &models.EventMetrics{
		TotalProcessed:   0, // Would be fetched from repository
		TotalFailed:      0,
		ProcessingRate:   0,
		AverageLatencyMs: 0,
	}, nil
}
