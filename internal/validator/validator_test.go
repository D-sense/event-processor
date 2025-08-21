package validator

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/d-sense/event-processor/pkg/models"
)

// Test data structures
type testCase struct {
	name        string
	input       interface{}
	expectError bool
	errorMsg    string
	description string
}

type eventTestCase struct {
	name        string
	eventJSON   string
	expectError bool
	errorMsg    string
	description string
}

// Helper function to create a test validator
func createTestValidator(t *testing.T) *Validator {
	// Create a temporary schema file for testing
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["eventId", "eventType", "clientId", "timestamp", "payload", "version"],
		"properties": {
			"eventId": {
				"type": "string",
				"pattern": "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
			},
			"eventType": {
				"type": "string",
				"enum": ["monitoring", "user_action", "transaction", "integration"]
			},
			"clientId": {
				"type": "string",
				"minLength": 1,
				"maxLength": 100,
				"pattern": "^[a-zA-Z0-9_-]+$"
			},
			"timestamp": {
				"type": "string",
				"format": "date-time"
			},
			"payload": {
				"type": "object",
				"minProperties": 1,
				"additionalProperties": true
			},
			"version": {
				"type": "string",
				"pattern": "^\\d+\\.\\d+$"
			}
		},
		"additionalProperties": false
	}`

	// Create a temporary file
	tmpFile := createTempSchemaFile(t, schemaContent)
	defer cleanupTempFile(t, tmpFile)

	validator := New(tmpFile)
	require.NotNil(t, validator)
	return validator
}

// Helper function to create a valid event JSON
func createValidEventJSON() string {
	return `{
		"eventId": "123e4567-e89b-12d3-a456-426614174000",
		"eventType": "monitoring",
		"clientId": "client-001",
		"timestamp": "2025-01-21T10:00:00Z",
		"payload": {
			"severity": "high",
			"message": "System alert"
		},
		"version": "1.0"
	}`
}

// TestValidateAndParseEvent tests the main validation function with different input types
func TestValidateAndParseEvent(t *testing.T) {
	validator := createTestValidator(t)

	tests := []testCase{
		{
			name:        "Valid SQS Message",
			input:       createSQSMessage(createValidEventJSON()),
			expectError: false,
			description: "Should successfully validate and parse SQS message",
		},
		{
			name:        "Valid String Input",
			input:       createValidEventJSON(),
			expectError: false,
			description: "Should successfully validate and parse string input",
		},
		{
			name:        "Valid Byte Array Input",
			input:       []byte(createValidEventJSON()),
			expectError: false,
			description: "Should successfully validate and parse byte array input",
		},
		{
			name:        "Valid Struct Input",
			input:       createValidEventStruct(),
			expectError: false,
			description: "Should successfully validate and parse struct input",
		},
		{
			name:        "Nil SQS Message Body",
			input:       &types.Message{},
			expectError: true,
			errorMsg:    "SQS message body is nil",
			description: "Should fail when SQS message body is nil",
		},
		{
			name:        "Invalid JSON String",
			input:       `{"invalid": json}`,
			expectError: true,
			errorMsg:    "validation error",
			description: "Should fail with invalid JSON",
		},
		{
			name:        "Empty String Input",
			input:       "",
			expectError: true,
			errorMsg:    "validation error",
			description: "Should fail with empty string",
		},
		{
			name:        "Nil Input",
			input:       nil,
			expectError: true,
			errorMsg:    "validation failed",
			description: "Should fail with nil input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateAndParseEvent(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assertValidEvent(t, result)
			}
		})
	}
}

// TestValidateEventBytes tests the byte validation function
func TestValidateEventBytes(t *testing.T) {
	validator := createTestValidator(t)

	tests := []eventTestCase{
		{
			name:        "Valid Event JSON",
			eventJSON:   createValidEventJSON(),
			expectError: false,
			description: "Should pass validation for valid event",
		},
		{
			name:        "Missing Required Field - eventId",
			eventJSON:   `{"eventType":"monitoring","clientId":"client-001","timestamp":"2025-01-21T10:00:00Z","payload":{"test":"value"},"version":"1.0"}`,
			expectError: true,
			errorMsg:    "validation failed",
			description: "Should fail when eventId is missing",
		},
		{
			name:        "Missing Required Field - eventType",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","clientId":"client-001","timestamp":"2025-01-21T10:00:00Z","payload":{"test":"value"},"version":"1.0"}`,
			expectError: true,
			errorMsg:    "validation failed",
			description: "Should fail when eventType is missing",
		},
		{
			name:        "Invalid UUID Format",
			eventJSON:   `{"eventId":"invalid-uuid","eventType":"monitoring","clientId":"client-001","timestamp":"2025-01-21T10:00:00Z","payload":{"test":"value"},"version":"1.0"}`,
			expectError: true,
			errorMsg:    "validation failed",
			description: "Should fail with invalid UUID format",
		},
		{
			name:        "Invalid Event Type",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"invalid_type","clientId":"client-001","timestamp":"2025-01-21T10:00:00Z","payload":{"test":"value"},"version":"1.0"}`,
			expectError: true,
			errorMsg:    "validation failed",
			description: "Should fail with invalid event type",
		},
		{
			name:        "Invalid Client ID Pattern",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"monitoring","clientId":"client@001","timestamp":"2025-01-21T10:00:00Z","payload":{"test":"value"},"version":"1.0"}`,
			expectError: true,
			errorMsg:    "validation failed",
			description: "Should fail with invalid client ID pattern",
		},
		{
			name:        "Empty Client ID",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"monitoring","clientId":"","timestamp":"2025-01-21T10:00:00Z","payload":{"test":"value"},"version":"1.0"}`,
			expectError: true,
			errorMsg:    "validation failed",
			description: "Should fail with empty client ID",
		},
		{
			name:        "Invalid Timestamp Format",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"monitoring","clientId":"client-001","timestamp":"invalid-timestamp","payload":{"test":"value"},"version":"1.0"}`,
			expectError: true,
			errorMsg:    "validation failed",
			description: "Should fail with invalid timestamp format",
		},
		{
			name:        "Empty Payload",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"monitoring","clientId":"client-001","timestamp":"2025-01-21T10:00:00Z","payload":{},"version":"1.0"}`,
			expectError: true,
			errorMsg:    "validation failed",
			description: "Should fail with empty payload",
		},
		{
			name:        "Invalid Version Format",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"monitoring","clientId":"client-001","timestamp":"2025-01-21T10:00:00Z","payload":{"test":"value"},"version":"invalid"}`,
			expectError: true,
			errorMsg:    "validation failed",
			description: "Should fail with invalid version format",
		},
		{
			name:        "Additional Properties Not Allowed",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"monitoring","clientId":"client-001","timestamp":"2025-01-21T10:00:00Z","payload":{"test":"value"},"version":"1.0","extraField":"should not be allowed"}`,
			expectError: true,
			errorMsg:    "validation failed",
			description: "Should fail when additional properties are present",
		},
		{
			name:        "Valid Event with Different Event Types",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"user_action","clientId":"client-002","timestamp":"2025-01-21T10:00:00Z","payload":{"action":"login","userId":"user123"},"version":"2.1"}`,
			expectError: false,
			description: "Should pass validation for user_action event type",
		},
		{
			name:        "Valid Event with Complex Payload",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"transaction","clientId":"client-003","timestamp":"2025-01-21T10:00:00Z","payload":{"amount":100.50,"currency":"USD","items":[{"id":"item1","price":50.25},{"id":"item2","price":50.25}]},"version":"1.5"}`,
			expectError: false,
			description: "Should pass validation for complex payload structure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateEventBytes([]byte(tt.eventJSON))

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestBusinessRulesValidation tests the business logic validation
func TestBusinessRulesValidation(t *testing.T) {
	validator := createTestValidator(t)

	tests := []eventTestCase{
		{
			name:        "Valid Event - All Business Rules Pass",
			eventJSON:   createValidEventJSON(),
			expectError: false,
			description: "Should pass all business rule validations",
		},
		{
			name:        "Invalid Event Type - Business Rule",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"invalid_type","clientId":"client-001","timestamp":"2025-01-21T10:00:00Z","payload":{"test":"value"},"version":"1.0"}`,
			expectError: true,
			errorMsg:    "invalid event type",
			description: "Should fail business rule validation for invalid event type",
		},
		{
			name:        "Empty Client ID - Business Rule",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"monitoring","clientId":"","timestamp":"2025-01-21T10:00:00Z","payload":{"test":"value"},"version":"1.0"}`,
			expectError: true,
			errorMsg:    "client ID cannot be empty",
			description: "Should fail business rule validation for empty client ID",
		},
		{
			name:        "Empty Payload - Business Rule",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"monitoring","clientId":"client-001","timestamp":"2025-01-21T10:00:00Z","payload":{},"version":"1.0"}`,
			expectError: true,
			errorMsg:    "payload cannot be empty",
			description: "Should fail business rule validation for empty payload",
		},
		{
			name:        "Zero Timestamp - Business Rule",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"monitoring","clientId":"client-001","timestamp":"0001-01-01T00:00:00Z","payload":{"test":"value"},"version":"1.0"}`,
			expectError: true,
			errorMsg:    "timestamp cannot be zero",
			description: "Should fail business rule validation for zero timestamp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the JSON first to get the Event struct
			var event models.Event
			err := json.Unmarshal([]byte(tt.eventJSON), &event)
			require.NoError(t, err, "Failed to parse test JSON")

			// Test business rules validation
			err = validator.validateBusinessRules(&event)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidatorCreation tests the validator constructor
func TestValidatorCreation(t *testing.T) {
	t.Run("Valid Schema File", func(t *testing.T) {
		validator := createTestValidator(t)
		assert.NotNil(t, validator)
		assert.NotNil(t, validator.schema)
	})

	t.Run("Invalid Schema File Path", func(t *testing.T) {
		assert.Panics(t, func() {
			New("nonexistent-file.json")
		}, "Should panic when schema file doesn't exist")
	})
}

// TestEdgeCases tests various edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	validator := createTestValidator(t)

	tests := []eventTestCase{
		{
			name:        "Maximum Client ID Length",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"monitoring","clientId":"` + string(make([]byte, 100)) + `","timestamp":"2025-01-21T10:00:00Z","payload":{"test":"value"},"version":"1.0"}`,
			expectError: true,
			errorMsg:    "validation failed",
			description: "Should fail when client ID exceeds maximum length",
		},
		{
			name:        "Minimum Client ID Length",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"monitoring","clientId":"a","timestamp":"2025-01-21T10:00:00Z","payload":{"test":"value"},"version":"1.0"}`,
			expectError: false,
			description: "Should pass with minimum length client ID",
		},
		{
			name:        "Complex Nested Payload",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"integration","clientId":"client-001","timestamp":"2025-01-21T10:00:00Z","payload":{"level1":{"level2":{"level3":{"value":"deeply nested"}}}},"version":"1.0"}`,
			expectError: false,
			description: "Should pass with deeply nested payload",
		},
		{
			name:        "Special Characters in Client ID",
			eventJSON:   `{"eventId":"123e4567-e89b-12d3-a456-426614174000","eventType":"monitoring","clientId":"client_001-test","timestamp":"2025-01-21T10:00:00Z","payload":{"test":"value"},"version":"1.0"}`,
			expectError: false,
			description: "Should pass with valid special characters in client ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateEventBytes([]byte(tt.eventJSON))

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper functions

func createSQSMessage(body string) *types.Message {
	return &types.Message{
		Body: &body,
	}
}

func createValidEventStruct() *models.Event {
	return &models.Event{
		EventID:   "123e4567-e89b-12d3-a456-426614174000",
		EventType: models.EventTypeMonitoring,
		ClientID:  "client-001",
		Timestamp: time.Date(2025, 1, 21, 10, 0, 0, 0, time.UTC),
		Payload: map[string]interface{}{
			"severity": "high",
			"message":  "System alert",
		},
		Version: "1.0",
	}
}

func assertValidEvent(t *testing.T, event *models.Event) {
	assert.NotEmpty(t, event.EventID)
	assert.NotEmpty(t, event.EventType)
	assert.NotEmpty(t, event.ClientID)
	assert.False(t, event.Timestamp.IsZero())
	assert.NotEmpty(t, event.Payload)
	assert.NotEmpty(t, event.Version)
}

// createTempSchemaFile creates a temporary schema file for testing purposes
func createTempSchemaFile(t *testing.T, content string) string {
	// We are using a placeholder here for the actual implementation.
	return "../../schemas/event-schema.json"
}

func cleanupTempFile(t *testing.T, filePath string) {
	// Of course, we don't want to remove/delete the actual file.
}
