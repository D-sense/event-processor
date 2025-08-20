package validator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/xeipuuv/gojsonschema"

	"github.com/d-sense/event-processor/pkg/models"
)

type Validator struct {
	schema *gojsonschema.Schema
}

// New creates a new validator with the provided schema file
func New(schemaPath string) *Validator {
	// Load schema file
	schemaBytes, err := ioutil.ReadFile(schemaPath)
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

// ValidateEvent validates an event against the JSON schema
func (v *Validator) ValidateEvent(eventData interface{}) error {
	// Convert to JSON for validation
	eventBytes, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

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

// ValidateAndParseEvent validates and parses event data into Event struct
func (v *Validator) ValidateAndParseEvent(eventData interface{}) (*models.Event, error) {
	// First validate against schema
	if err := v.ValidateEvent(eventData); err != nil {
		return nil, err
	}

	// Convert to JSON and then to Event struct
	eventBytes, err := json.Marshal(eventData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

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

// GetValidationResult returns detailed validation results
func (v *Validator) GetValidationResult(eventData interface{}) *models.ValidationResult {
	err := v.ValidateEvent(eventData)
	if err != nil {
		return &models.ValidationResult{
			Valid:  false,
			Errors: []string{err.Error()},
		}
	}

	return &models.ValidationResult{
		Valid:  true,
		Errors: nil,
	}
}
