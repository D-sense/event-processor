package models

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of event
type EventType string

const (
	EventTypeMonitoring  EventType = "monitoring"
	EventTypeUserAction  EventType = "user_action"
	EventTypeTransaction EventType = "transaction"
	EventTypeIntegration EventType = "integration"
)

// EventStatus represents the processing status of an event
type EventStatus string

const (
	EventStatusPending   EventStatus = "pending"
	EventStatusProcessed EventStatus = "processed"
	EventStatusFailed    EventStatus = "failed"
	EventStatusRetry     EventStatus = "retry"
)

// Event represents the core event structure
type Event struct {
	EventID   string                 `json:"eventId" dynamodb:"event_id"`
	EventType EventType              `json:"eventType" dynamodb:"event_type"`
	ClientID  string                 `json:"clientId" dynamodb:"client_id"`
	Timestamp time.Time              `json:"timestamp" dynamodb:"timestamp"`
	Payload   map[string]interface{} `json:"payload" dynamodb:"payload"`
	Version   string                 `json:"version" dynamodb:"version"`
}

// ProcessedEvent represents an event after processing
type ProcessedEvent struct {
	Event
	ProcessedAt time.Time   `json:"processedAt" dynamodb:"processed_at"`
	Status      EventStatus `json:"status" dynamodb:"status"`
	ErrorMsg    string      `json:"errorMsg,omitempty" dynamodb:"error_msg,omitempty"`
	RetryCount  int         `json:"retryCount" dynamodb:"retry_count"`
	TTL         int64       `json:"ttl" dynamodb:"ttl"`
}

// NewEvent creates a new event with generated ID and timestamp
func NewEvent(eventType EventType, clientID string, payload map[string]interface{}, version string) *Event {
	return &Event{
		EventID:   uuid.New().String(),
		EventType: eventType,
		ClientID:  clientID,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
		Version:   version,
	}
}

// ToProcessedEvent converts an Event to ProcessedEvent
func (e *Event) ToProcessedEvent() *ProcessedEvent {
	return &ProcessedEvent{
		Event:       *e,
		ProcessedAt: time.Now().UTC(),
		Status:      EventStatusPending,
		RetryCount:  0,
		TTL:         time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 days TTL
	}
}

// IsValidEventType checks if the event type is valid
func IsValidEventType(eventType string) bool {
	switch EventType(eventType) {
	case EventTypeMonitoring, EventTypeUserAction, EventTypeTransaction, EventTypeIntegration:
		return true
	default:
		return false
	}
}

// EventMetrics represents metrics for event processing
type EventMetrics struct {
	TotalProcessed   int64 `json:"totalProcessed"`
	TotalFailed      int64 `json:"totalFailed"`
	ProcessingRate   int64 `json:"processingRate"`
	AverageLatencyMs int64 `json:"averageLatencyMs"`
}

// ClientConfig represents per-client configuration
type ClientConfig struct {
	ClientID     string            `json:"clientId" dynamodb:"client_id"`
	AllowedTypes []EventType       `json:"allowedTypes" dynamodb:"allowed_types"`
	Config       map[string]string `json:"config" dynamodb:"config"`
	Active       bool              `json:"active" dynamodb:"active"`
}

// ValidationResult represents the result of event validation
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}
