package processor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/d-sense/event-processor/pkg/models"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) SaveEvent(ctx context.Context, event *models.ProcessedEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockRepository) GetClientConfig(ctx context.Context, clientID string) (*models.ClientConfig, error) {
	args := m.Called(ctx, clientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ClientConfig), args.Error(1)
}

func (m *MockRepository) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockValidator is a mock implementation of the Validator
type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) ValidateAndParseEvent(eventData interface{}) (*models.Event, error) {
	args := m.Called(eventData)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Event), args.Error(1)
}

// Test data structures
type processEventTestCase struct {
	name           string
	eventData      interface{}
	mockValidator  func(*MockValidator)
	mockRepository func(*MockRepository)
	expectError    bool
	errorMsg       string
	description    string
}

type triageEventTestCase struct {
	name           string
	event          *models.Event
	mockRepository func(*MockRepository)
	expectError    bool
	errorMsg       string
	expectedStatus models.EventStatus
	description    string
}

type eventProcessingTestCase struct {
	name           string
	event          *models.Event
	expectedError  bool
	errorMsg       string
	expectedFields map[string]interface{}
	description    string
}

// TestProcessEvent tests the main ProcessEvent function
func TestProcessEvent(t *testing.T) {
	tests := []processEventTestCase{
		{
			name:      "Valid Event - Successful Processing",
			eventData: "valid-event-data",
			mockValidator: func(mv *MockValidator) {
				mv.On("ValidateAndParseEvent", "valid-event-data").Return(createValidEvent(), nil)
			},
			mockRepository: func(mr *MockRepository) {
				mr.On("GetClientConfig", mock.Anything, "client-001").Return(createValidClientConfig(), nil)
				mr.On("SaveEvent", mock.Anything, mock.AnythingOfType("*models.ProcessedEvent")).Return(nil)
			},
			expectError: false,
			description: "Should successfully process a valid event",
		},
		{
			name:      "Validation Failure",
			eventData: "invalid-event-data",
			mockValidator: func(mv *MockValidator) {
				mv.On("ValidateAndParseEvent", "invalid-event-data").Return(nil, errors.New("validation error"))
			},
			mockRepository: func(mr *MockRepository) {
				// No repository calls expected since validation fails
			},
			expectError: true,
			errorMsg:    "validation failed",
			description: "Should fail when event validation fails",
		},
		{
			name:      "Triage Failure - Client Permission Denied",
			eventData: "valid-event-data",
			mockValidator: func(mv *MockValidator) {
				mv.On("ValidateAndParseEvent", "valid-event-data").Return(createValidEvent(), nil)
			},
			mockRepository: func(mr *MockRepository) {
				// Mock client with restricted permissions
				mr.On("GetClientConfig", mock.Anything, "client-001").Return(createRestrictedClientConfig(), nil)
			},
			expectError: true,
			errorMsg:    "triage failed",
			description: "Should fail when client lacks permission for event type",
		},
		{
			name:      "Persistence Failure",
			eventData: "valid-event-data",
			mockValidator: func(mv *MockValidator) {
				mv.On("ValidateAndParseEvent", "valid-event-data").Return(createValidEvent(), nil)
			},
			mockRepository: func(mr *MockRepository) {
				// Mock successful triage but failed persistence
				mr.On("GetClientConfig", mock.Anything, "client-001").Return(createValidClientConfig(), nil)
				mr.On("SaveEvent", mock.Anything, mock.AnythingOfType("*models.ProcessedEvent")).Return(errors.New("persistence error"))
			},
			expectError: true,
			errorMsg:    "persistence failed",
			description: "Should fail when event persistence fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockRepo := &MockRepository{}
			mockVal := &MockValidator{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockValidator != nil {
				tt.mockValidator(mockVal)
			}
			if tt.mockRepository != nil {
				tt.mockRepository(mockRepo)
			}

			// Create processor with mock validator
			processor := &EventProcessor{
				repository: mockRepo,
				validator:  mockVal,
				logger:     logger,
			}

			// Execute test
			err := processor.ProcessEvent(context.Background(), tt.eventData)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify mocks
			mockRepo.AssertExpectations(t)
			mockVal.AssertExpectations(t)
		})
	}
}

// TestTriageEvent tests the event triage logic
func TestTriageEvent(t *testing.T) {
	tests := []triageEventTestCase{
		{
			name:  "Valid Monitoring Event",
			event: createMonitoringEvent(),
			mockRepository: func(mr *MockRepository) {
				mr.On("GetClientConfig", mock.Anything, "client-001").Return(createValidClientConfig(), nil)
			},
			expectError:    false,
			expectedStatus: models.EventStatusProcessed,
			description:    "Should successfully triage monitoring event",
		},
		{
			name:  "Valid User Action Event",
			event: createUserActionEvent(),
			mockRepository: func(mr *MockRepository) {
				mr.On("GetClientConfig", mock.Anything, "client-001").Return(createValidClientConfig(), nil)
			},
			expectError:    false,
			expectedStatus: models.EventStatusProcessed,
			description:    "Should successfully triage user action event",
		},
		{
			name:  "Valid Transaction Event",
			event: createTransactionEvent(),
			mockRepository: func(mr *MockRepository) {
				mr.On("GetClientConfig", mock.Anything, "client-001").Return(createValidClientConfig(), nil)
			},
			expectError:    false,
			expectedStatus: models.EventStatusProcessed,
			description:    "Should successfully triage transaction event",
		},
		{
			name:  "Valid Integration Event",
			event: createIntegrationEvent(),
			mockRepository: func(mr *MockRepository) {
				mr.On("GetClientConfig", mock.Anything, "client-001").Return(createValidClientConfig(), nil)
			},
			expectError:    false,
			expectedStatus: models.EventStatusProcessed,
			description:    "Should successfully triage integration event",
		},
		{
			name:  "Unknown Event Type",
			event: createUnknownEventType(),
			mockRepository: func(mr *MockRepository) {
				// Even unknown event types go through client permission validation
				mr.On("GetClientConfig", mock.Anything, "client-001").Return(createValidClientConfig(), nil)
			},
			expectError:    true,
			errorMsg:       "client permission validation failed",
			expectedStatus: models.EventStatusFailed,
			description:    "Should fail when unknown event type is not allowed by client",
		},
		{
			name:  "Client Permission Denied",
			event: createValidEvent(),
			mockRepository: func(mr *MockRepository) {
				mr.On("GetClientConfig", mock.Anything, "client-001").Return(createRestrictedClientConfig(), nil)
			},
			expectError: true,
			errorMsg:    "client permission validation failed",
			description: "Should fail when client lacks permission for event type",
		},
		{
			name:  "Inactive Client",
			event: createValidEvent(),
			mockRepository: func(mr *MockRepository) {
				mr.On("GetClientConfig", mock.Anything, "client-001").Return(createInactiveClientConfig(), nil)
			},
			expectError: true,
			errorMsg:    "client permission validation failed",
			description: "Should fail when client is inactive",
		},
		{
			name:  "Client Config Not Found - Allow by Default",
			event: createValidEvent(),
			mockRepository: func(mr *MockRepository) {
				mr.On("GetClientConfig", mock.Anything, "client-001").Return(nil, errors.New("not found"))
			},
			expectError:    false,
			expectedStatus: models.EventStatusProcessed,
			description:    "Should allow event when client config not found (default behavior)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockRepo := &MockRepository{}
			mockVal := &MockValidator{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockRepository != nil {
				tt.mockRepository(mockRepo)
			}

			// Create processor with mock validator
			processor := &EventProcessor{
				repository: mockRepo,
				validator:  mockVal,
				logger:     logger,
			}

			// Execute test
			result, err := processor.triageEvent(context.Background(), tt.event, logger.WithField("test", "triage"))

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedStatus, result.Status)
			}

			// Verify mocks
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestProcessMonitoringEvent tests monitoring event processing
func TestProcessMonitoringEvent(t *testing.T) {
	tests := []eventProcessingTestCase{
		{
			name:          "High Severity Event",
			event:         createHighSeverityMonitoringEvent(),
			expectedError: false,
			expectedFields: map[string]interface{}{
				"priority": "high",
			},
			description: "Should add priority flag for high-severity events",
		},
		{
			name:          "Low Severity Event",
			event:         createLowSeverityMonitoringEvent(),
			expectedError: false,
			expectedFields: map[string]interface{}{
				"priority": nil, // Should not have priority flag
			},
			description: "Should not add priority flag for low-severity events",
		},
		{
			name:          "Critical Severity Event",
			event:         createCriticalSeverityMonitoringEvent(),
			expectedError: false,
			expectedFields: map[string]interface{}{
				"priority": "high",
			},
			description: "Should add priority flag for critical-severity events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create processor
			processor := &EventProcessor{}
			logger := logrus.New().WithField("test", "monitoring")

			// Create processed event
			processedEvent := tt.event.ToProcessedEvent()

			// Execute test
			err := processor.processMonitoringEvent(tt.event, processedEvent, logger)

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Check expected fields
				for field, expectedValue := range tt.expectedFields {
					if expectedValue == nil {
						assert.NotContains(t, processedEvent.Payload, field)
					} else {
						assert.Equal(t, expectedValue, processedEvent.Payload[field])
					}
				}
			}
		})
	}
}

// TestProcessUserActionEvent tests user action event processing
func TestProcessUserActionEvent(t *testing.T) {
	tests := []eventProcessingTestCase{
		{
			name:          "Valid User Action Event",
			event:         createUserActionEvent(),
			expectedError: false,
			expectedFields: map[string]interface{}{
				"processedAt": mock.AnythingOfType("string"),
			},
			description: "Should successfully process user action event with audit timestamp",
		},
		{
			name:          "Missing User ID",
			event:         createUserActionEventMissingField("userId"),
			expectedError: true,
			errorMsg:      "missing required field for user action: userId",
			description:   "Should fail when userId is missing",
		},
		{
			name:          "Missing Action",
			event:         createUserActionEventMissingField("action"),
			expectedError: true,
			errorMsg:      "missing required field for user action: action",
			description:   "Should fail when action is missing",
		},
		{
			name:          "Missing Resource",
			event:         createUserActionEventMissingField("resource"),
			expectedError: true,
			errorMsg:      "missing required field for user action: resource",
			description:   "Should fail when resource is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create processor
			processor := &EventProcessor{}
			logger := logrus.New().WithField("test", "user_action")

			// Create processed event
			processedEvent := tt.event.ToProcessedEvent()

			// Execute test
			err := processor.processUserActionEvent(tt.event, processedEvent, logger)

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				// Check expected fields
				for field, expectedValue := range tt.expectedFields {
					if field == "processedAt" {
						assert.Contains(t, processedEvent.Payload, field)
						// Verify it's a valid timestamp
						timestampStr := processedEvent.Payload[field].(string)
						_, parseErr := time.Parse(time.RFC3339, timestampStr)
						assert.NoError(t, parseErr)
					} else {
						assert.Equal(t, expectedValue, processedEvent.Payload[field])
					}
				}
			}
		})
	}
}

// TestProcessTransactionEvent tests transaction event processing
func TestProcessTransactionEvent(t *testing.T) {
	tests := []eventProcessingTestCase{
		{
			name:          "Valid Transaction Event",
			event:         createTransactionEvent(),
			expectedError: false,
			expectedFields: map[string]interface{}{
				"highValue": nil, // Should not have highValue flag for amount < 10000
			},
			description: "Should successfully process transaction event",
		},
		{
			name:          "High Value Transaction",
			event:         createHighValueTransactionEvent(),
			expectedError: false,
			expectedFields: map[string]interface{}{
				"highValue": true,
			},
			description: "Should flag high-value transactions (>10000)",
		},
		{
			name:          "Missing Transaction ID",
			event:         createTransactionEventMissingField("transactionId"),
			expectedError: true,
			errorMsg:      "missing required field for transaction: transactionId",
			description:   "Should fail when transactionId is missing",
		},
		{
			name:          "Missing Amount",
			event:         createTransactionEventMissingField("amount"),
			expectedError: true,
			errorMsg:      "missing required field for transaction: amount",
			description:   "Should fail when amount is missing",
		},
		{
			name:          "Missing Currency",
			event:         createTransactionEventMissingField("currency"),
			expectedError: true,
			errorMsg:      "missing required field for transaction: currency",
			description:   "Should fail when currency is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create processor
			processor := &EventProcessor{}
			logger := logrus.New().WithField("test", "transaction")

			// Create processed event
			processedEvent := tt.event.ToProcessedEvent()

			// Execute test
			err := processor.processTransactionEvent(tt.event, processedEvent, logger)

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				// Check expected fields
				for field, expectedValue := range tt.expectedFields {
					if expectedValue == nil {
						assert.NotContains(t, processedEvent.Payload, field)
					} else {
						assert.Equal(t, expectedValue, processedEvent.Payload[field])
					}
				}
			}
		})
	}
}

// TestProcessIntegrationEvent tests integration event processing
func TestProcessIntegrationEvent(t *testing.T) {
	tests := []eventProcessingTestCase{
		{
			name:          "Valid Integration Event",
			event:         createIntegrationEvent(),
			expectedError: false,
			expectedFields: map[string]interface{}{
				"integrationProcessedAt": mock.AnythingOfType("string"),
			},
			description: "Should successfully process integration event with timestamp",
		},
		{
			name:          "Missing Source",
			event:         createIntegrationEventMissingField("source"),
			expectedError: true,
			errorMsg:      "missing required field for integration: source",
			description:   "Should fail when source is missing",
		},
		{
			name:          "Missing Target",
			event:         createIntegrationEventMissingField("target"),
			expectedError: true,
			errorMsg:      "missing required field for integration: target",
			description:   "Should fail when target is missing",
		},
		{
			name:          "Missing Operation",
			event:         createIntegrationEventMissingField("operation"),
			expectedError: true,
			errorMsg:      "missing required field for integration: operation",
			description:   "Should fail when operation is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create processor
			processor := &EventProcessor{}
			logger := logrus.New().WithField("test", "integration")

			// Create processed event
			processedEvent := tt.event.ToProcessedEvent()

			// Execute test
			err := processor.processIntegrationEvent(tt.event, processedEvent, logger)

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				// Check expected fields
				for field, expectedValue := range tt.expectedFields {
					if field == "integrationProcessedAt" {
						assert.Contains(t, processedEvent.Payload, field)
						// Verify it's a valid timestamp
						timestampStr := processedEvent.Payload[field].(string)
						_, parseErr := time.Parse(time.RFC3339, timestampStr)
						assert.NoError(t, parseErr)
					} else {
						assert.Equal(t, expectedValue, processedEvent.Payload[field])
					}
				}
			}
		})
	}
}

// TestValidateClientPermissions tests client permission validation
func TestValidateClientPermissions(t *testing.T) {
	tests := []struct {
		name           string
		clientID       string
		eventType      models.EventType
		mockRepository func(*MockRepository)
		expectError    bool
		errorMsg       string
		description    string
	}{
		{
			name:      "Valid Client with Permission",
			clientID:  "client-001",
			eventType: models.EventTypeMonitoring,
			mockRepository: func(mr *MockRepository) {
				mr.On("GetClientConfig", mock.Anything, "client-001").Return(createValidClientConfig(), nil)
			},
			expectError: false,
			description: "Should allow event when client has permission",
		},
		{
			name:      "Client Without Permission",
			clientID:  "client-002",
			eventType: models.EventTypeMonitoring,
			mockRepository: func(mr *MockRepository) {
				mr.On("GetClientConfig", mock.Anything, "client-002").Return(createRestrictedClientConfig(), nil)
			},
			expectError: true,
			errorMsg:    "client client-002 is not allowed to send events of type monitoring",
			description: "Should deny event when client lacks permission",
		},
		{
			name:      "Inactive Client",
			clientID:  "client-003",
			eventType: models.EventTypeUserAction,
			mockRepository: func(mr *MockRepository) {
				mr.On("GetClientConfig", mock.Anything, "client-003").Return(createInactiveClientConfig(), nil)
			},
			expectError: true,
			errorMsg:    "client client-003 is not active",
			description: "Should deny event when client is inactive",
		},
		{
			name:      "Client Config Not Found - Allow by Default",
			clientID:  "client-004",
			eventType: models.EventTypeTransaction,
			mockRepository: func(mr *MockRepository) {
				mr.On("GetClientConfig", mock.Anything, "client-004").Return(nil, errors.New("not found"))
			},
			expectError: false,
			description: "Should allow event when client config not found (default behavior)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockRepo := &MockRepository{}
			logger := logrus.New().WithField("test", "permissions")

			// Setup mocks
			if tt.mockRepository != nil {
				tt.mockRepository(mockRepo)
			}

			// Create processor
			processor := &EventProcessor{repository: mockRepo}

			// Execute test
			err := processor.validateClientPermissions(context.Background(), tt.clientID, tt.eventType, logger)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify mocks
			mockRepo.AssertExpectations(t)
		})
	}
}

// Helper functions to create test data

func createValidEvent() *models.Event {
	return &models.Event{
		EventID:   "123e4567-e89b-12d3-a456-426614174000",
		EventType: models.EventTypeMonitoring,
		ClientID:  "client-001",
		Timestamp: time.Now().UTC(),
		Payload: map[string]interface{}{
			"severity": "medium",
			"message":  "Test event",
		},
		Version: "1.0",
	}
}

func createMonitoringEvent() *models.Event {
	event := createValidEvent()
	event.EventType = models.EventTypeMonitoring
	return event
}

func createHighSeverityMonitoringEvent() *models.Event {
	event := createValidEvent()
	event.EventType = models.EventTypeMonitoring
	event.Payload["severity"] = "high"
	return event
}

func createCriticalSeverityMonitoringEvent() *models.Event {
	event := createValidEvent()
	event.EventType = models.EventTypeMonitoring
	event.Payload["severity"] = "critical"
	return event
}

func createLowSeverityMonitoringEvent() *models.Event {
	event := createValidEvent()
	event.EventType = models.EventTypeMonitoring
	event.Payload["severity"] = "low"
	return event
}

func createUserActionEvent() *models.Event {
	event := createValidEvent()
	event.EventType = models.EventTypeUserAction
	event.Payload = map[string]interface{}{
		"userId":   "user123",
		"action":   "login",
		"resource": "dashboard",
	}
	return event
}

func createUserActionEventMissingField(field string) *models.Event {
	event := createUserActionEvent()
	delete(event.Payload, field)
	return event
}

func createTransactionEvent() *models.Event {
	event := createValidEvent()
	event.EventType = models.EventTypeTransaction
	event.Payload = map[string]interface{}{
		"transactionId": "txn123",
		"amount":        5000.0,
		"currency":      "USD",
	}
	return event
}

func createHighValueTransactionEvent() *models.Event {
	event := createValidEvent()
	event.EventType = models.EventTypeTransaction
	event.Payload = map[string]interface{}{
		"transactionId": "txn456",
		"amount":        15000.0,
		"currency":      "USD",
	}
	return event
}

func createTransactionEventMissingField(field string) *models.Event {
	event := createTransactionEvent()
	delete(event.Payload, field)
	return event
}

func createIntegrationEvent() *models.Event {
	event := createValidEvent()
	event.EventType = models.EventTypeIntegration
	event.Payload = map[string]interface{}{
		"source":    "system-a",
		"target":    "system-b",
		"operation": "sync",
	}
	return event
}

func createIntegrationEventMissingField(field string) *models.Event {
	event := createIntegrationEvent()
	delete(event.Payload, field)
	return event
}

func createUnknownEventType() *models.Event {
	event := createValidEvent()
	event.EventType = "unknown_type"
	return event
}

func createValidClientConfig() *models.ClientConfig {
	return &models.ClientConfig{
		ClientID: "client-001",
		AllowedTypes: []models.EventType{
			models.EventTypeMonitoring,
			models.EventTypeUserAction,
			models.EventTypeTransaction,
			models.EventTypeIntegration,
		},
		Active: true,
		Config: map[string]string{
			"rate_limit": "1000",
		},
	}
}

func createRestrictedClientConfig() *models.ClientConfig {
	return &models.ClientConfig{
		ClientID: "client-002",
		AllowedTypes: []models.EventType{
			models.EventTypeUserAction,
			models.EventTypeTransaction,
		},
		Active: true,
		Config: map[string]string{
			"rate_limit": "500",
		},
	}
}

func createInactiveClientConfig() *models.ClientConfig {
	return &models.ClientConfig{
		ClientID: "client-003",
		AllowedTypes: []models.EventType{
			models.EventTypeMonitoring,
			models.EventTypeUserAction,
			models.EventTypeTransaction,
			models.EventTypeIntegration,
		},
		Active: false,
		Config: map[string]string{
			"rate_limit": "1000",
		},
	}
}
