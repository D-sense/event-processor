package validator

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/xeipuuv/gojsonschema"

	"github.com/d-sense/event-processor/pkg/models"
)

type Validator struct {
	schema *gojsonschema.Schema
}

// New creates a new validator with the provided schema file
func New(schemaPath string) *Validator {
	// Load schema file
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load schema file: %v", err))
	}

	// Compile schema
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		panic(fmt.Sprintf("Failed to compile schema: %v", err))
	}

	return &Validator{
		schema: schema,
	}
}

// ValidateAndParseEvent validates and parses event data into Event struct
func (v *Validator) ValidateAndParseEvent(eventData interface{}) (*models.Event, error) {
	var eventBytes []byte
	var err error

	// Handle different input types
	switch data := eventData.(type) {
	case *types.Message:
		// SQS message - extract the body
		if data.Body == nil {
			return nil, fmt.Errorf("SQS message body is nil")
		}
		eventBytes = []byte(*data.Body)
	case string:
		// String data
		eventBytes = []byte(data)
	case []byte:
		// Byte data
		eventBytes = data
	default:
		// Try to marshal to JSON first
		eventBytes, err = json.Marshal(eventData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal event data: %w", err)
		}
	}

	// First validate against schema
	if err := v.ValidateEventBytes(eventBytes); err != nil {
		return nil, err
	}

	// Parse into Event struct
	var event models.Event
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	// Additional business logic validation
	if err := v.validateBusinessRules(&event); err != nil {
		return nil, err
	}

	return &event, nil
}

// ValidateEventBytes validates event bytes against the JSON schema
func (v *Validator) ValidateEventBytes(eventBytes []byte) error {
	// Create document loader
	documentLoader := gojsonschema.NewBytesLoader(eventBytes)

	// Validate against schema
	result, err := v.schema.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		var errors []string
		for _, err := range result.Errors() {
			errors = append(errors, err.String())
		}
		return fmt.Errorf("validation failed: %v", errors)
	}

	return nil
}

// validateBusinessRules performs additional business logic validation
func (v *Validator) validateBusinessRules(event *models.Event) error {
	// Validate event type
	if !models.IsValidEventType(string(event.EventType)) {
		return fmt.Errorf("invalid event type: %s", event.EventType)
	}

	// Validate client ID format
	if len(event.ClientID) == 0 {
		return fmt.Errorf("client ID cannot be empty")
	}

	// Validate payload is not empty
	if len(event.Payload) == 0 {
		return fmt.Errorf("payload cannot be empty")
	}

	// Validate timestamp is not zero
	if event.Timestamp.IsZero() {
		return fmt.Errorf("timestamp cannot be zero")
	}

	return nil
}
