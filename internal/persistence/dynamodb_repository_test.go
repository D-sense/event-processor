package persistence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/d-sense/event-processor/internal/config"
	"github.com/d-sense/event-processor/pkg/models"
)

// MockDynamoDBClient is a mock implementation of the DynamoDB client
type MockDynamoDBClient struct {
	mock.Mock
}

func (m *MockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.PutItemOutput), args.Error(1)
}

func (m *MockDynamoDBClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.GetItemOutput), args.Error(1)
}

func (m *MockDynamoDBClient) DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.DescribeTableOutput), args.Error(1)
}

func (m *MockDynamoDBClient) ListTables(ctx context.Context, params *dynamodb.ListTablesInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.ListTablesOutput), args.Error(1)
}

func (m *MockDynamoDBClient) CreateTable(ctx context.Context, params *dynamodb.CreateTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.CreateTableOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.CreateTableOutput), args.Error(1)
}

// Test data structures
type saveEventTestCase struct {
	name        string
	event       *models.ProcessedEvent
	mockClient  func(*MockDynamoDBClient)
	expectError bool
	errorMsg    string
	description string
}

type getClientConfigTestCase struct {
	name           string
	clientID       string
	mockClient     func(*MockDynamoDBClient)
	expectError    bool
	errorMsg       string
	expectedConfig *models.ClientConfig
	description    string
}

type healthCheckTestCase struct {
	name        string
	mockClient  func(*MockDynamoDBClient)
	expectError bool
	errorMsg    string
	description string
}

type marshalPayloadTestCase struct {
	name           string
	payload        map[string]interface{}
	expectedResult map[string]types.AttributeValue
	description    string
}

// TestSaveEvent tests the SaveEvent method
func TestSaveEvent(t *testing.T) {
	tests := []saveEventTestCase{
		{
			name:  "Valid Event - Successful Save",
			event: createValidProcessedEvent(),
			mockClient: func(mc *MockDynamoDBClient) {
				mc.On("PutItem", mock.Anything, mock.AnythingOfType("*dynamodb.PutItemInput")).Return(&dynamodb.PutItemOutput{}, nil)
			},
			expectError: false,
			description: "Should successfully save a valid event",
		},
		{
			name:  "Event with Error Message",
			event: createProcessedEventWithError(),
			mockClient: func(mc *MockDynamoDBClient) {
				mc.On("PutItem", mock.Anything, mock.AnythingOfType("*dynamodb.PutItemInput")).Return(&dynamodb.PutItemOutput{}, nil)
			},
			expectError: false,
			description: "Should successfully save event with error message",
		},
		{
			name:  "Event with Complex Payload",
			event: createProcessedEventWithComplexPayload(),
			mockClient: func(mc *MockDynamoDBClient) {
				mc.On("PutItem", mock.Anything, mock.AnythingOfType("*dynamodb.PutItemInput")).Return(&dynamodb.PutItemOutput{}, nil)
			},
			expectError: false,
			description: "Should successfully save event with complex nested payload",
		},
		{
			name:  "DynamoDB PutItem Failure",
			event: createValidProcessedEvent(),
			mockClient: func(mc *MockDynamoDBClient) {
				mc.On("PutItem", mock.Anything, mock.AnythingOfType("*dynamodb.PutItemInput")).Return(nil, errors.New("dynamodb error"))
			},
			expectError: true,
			errorMsg:    "failed to save event to DynamoDB",
			description: "Should fail when DynamoDB PutItem operation fails",
		},
		{
			name:  "Event with Zero Timestamp",
			event: createProcessedEventWithZeroTimestamp(),
			mockClient: func(mc *MockDynamoDBClient) {
				mc.On("PutItem", mock.Anything, mock.AnythingOfType("*dynamodb.PutItemInput")).Return(&dynamodb.PutItemOutput{}, nil)
			},
			expectError: false,
			description: "Should handle zero timestamp gracefully",
		},
		{
			name:  "Event with Empty Payload",
			event: createProcessedEventWithEmptyPayload(),
			mockClient: func(mc *MockDynamoDBClient) {
				mc.On("PutItem", mock.Anything, mock.AnythingOfType("*dynamodb.PutItemInput")).Return(&dynamodb.PutItemOutput{}, nil)
			},
			expectError: false,
			description: "Should handle empty payload gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockDynamoDBClient{}

			// Setup mocks
			if tt.mockClient != nil {
				tt.mockClient(mockClient)
			}

			// Create repository with mock client
			repo := &DynamoDBRepository{
				client:    mockClient,
				tableName: "test-events",
			}

			// Execute test
			err := repo.SaveEvent(context.Background(), tt.event)

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
			mockClient.AssertExpectations(t)
		})
	}
}

// TestGetClientConfig tests the GetClientConfig method
func TestGetClientConfig(t *testing.T) {
	tests := []getClientConfigTestCase{
		{
			name:     "Valid Client Config - All Fields Present",
			clientID: "client-001",
			mockClient: func(mc *MockDynamoDBClient) {
				item := map[string]types.AttributeValue{
					"client_id":     &types.AttributeValueMemberS{Value: "client-001"},
					"allowed_types": &types.AttributeValueMemberSS{Value: []string{"monitoring", "user_action"}},
					"active":        &types.AttributeValueMemberBOOL{Value: true},
					"config": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
						"rate_limit": &types.AttributeValueMemberS{Value: "1000"},
					}},
				}
				mc.On("GetItem", mock.Anything, mock.AnythingOfType("*dynamodb.GetItemInput")).Return(&dynamodb.GetItemOutput{Item: item}, nil)
			},
			expectError: false,
			expectedConfig: &models.ClientConfig{
				ClientID:     "client-001",
				AllowedTypes: []models.EventType{models.EventTypeMonitoring, models.EventTypeUserAction},
				Active:       true,
				Config: map[string]string{
					"rate_limit": "1000",
				},
			},
			description: "Should successfully retrieve complete client configuration",
		},
		{
			name:     "Client Config - Minimal Fields",
			clientID: "client-002",
			mockClient: func(mc *MockDynamoDBClient) {
				item := map[string]types.AttributeValue{
					"client_id": &types.AttributeValueMemberS{Value: "client-002"},
				}
				mc.On("GetItem", mock.Anything, mock.AnythingOfType("*dynamodb.GetItemInput")).Return(&dynamodb.GetItemOutput{Item: item}, nil)
			},
			expectError: false,
			expectedConfig: &models.ClientConfig{
				ClientID:     "client-002",
				AllowedTypes: nil,
				Active:       true, // Default value
				Config:       nil,
			},
			description: "Should return default values for missing fields",
		},
		{
			name:     "Client Config - Inactive Client",
			clientID: "client-003",
			mockClient: func(mc *MockDynamoDBClient) {
				item := map[string]types.AttributeValue{
					"client_id": &types.AttributeValueMemberS{Value: "client-003"},
					"active":    &types.AttributeValueMemberBOOL{Value: false},
				}
				mc.On("GetItem", mock.Anything, mock.AnythingOfType("*dynamodb.GetItemInput")).Return(&dynamodb.GetItemOutput{Item: item}, nil)
			},
			expectError: false,
			expectedConfig: &models.ClientConfig{
				ClientID:     "client-003",
				AllowedTypes: nil,
				Active:       false,
				Config:       nil,
			},
			description: "Should correctly handle inactive client status",
		},
		{
			name:     "Client Config - Complex Allowed Types",
			clientID: "client-004",
			mockClient: func(mc *MockDynamoDBClient) {
				item := map[string]types.AttributeValue{
					"client_id": &types.AttributeValueMemberS{Value: "client-004"},
					"allowed_types": &types.AttributeValueMemberSS{Value: []string{
						"monitoring", "user_action", "transaction", "integration",
					}},
					"active": &types.AttributeValueMemberBOOL{Value: true},
				}
				mc.On("GetItem", mock.Anything, mock.AnythingOfType("*dynamodb.GetItemInput")).Return(&dynamodb.GetItemOutput{Item: item}, nil)
			},
			expectError: false,
			expectedConfig: &models.ClientConfig{
				ClientID: "client-004",
				AllowedTypes: []models.EventType{
					models.EventTypeMonitoring,
					models.EventTypeUserAction,
					models.EventTypeTransaction,
					models.EventTypeIntegration,
				},
				Active: true,
				Config: nil,
			},
			description: "Should handle all event types correctly",
		},
		{
			name:     "Client Config Not Found",
			clientID: "client-999",
			mockClient: func(mc *MockDynamoDBClient) {
				mc.On("GetItem", mock.Anything, mock.AnythingOfType("*dynamodb.GetItemInput")).Return(&dynamodb.GetItemOutput{Item: nil}, nil)
			},
			expectError:    true,
			errorMsg:       "client config not found",
			expectedConfig: nil,
			description:    "Should fail when client config doesn't exist",
		},
		{
			name:     "DynamoDB GetItem Failure",
			clientID: "client-001",
			mockClient: func(mc *MockDynamoDBClient) {
				mc.On("GetItem", mock.Anything, mock.AnythingOfType("*dynamodb.GetItemInput")).Return(nil, errors.New("dynamodb error"))
			},
			expectError:    true,
			errorMsg:       "failed to get client config",
			expectedConfig: nil,
			description:    "Should fail when DynamoDB GetItem operation fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockDynamoDBClient{}

			// Setup mocks
			if tt.mockClient != nil {
				tt.mockClient(mockClient)
			}

			// Create repository with mock client
			repo := &DynamoDBRepository{
				client:    mockClient,
				tableName: "test-events",
			}

			// Execute test
			result, err := repo.GetClientConfig(context.Background(), tt.clientID)

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
				assert.Equal(t, tt.expectedConfig.ClientID, result.ClientID)
				assert.Equal(t, tt.expectedConfig.Active, result.Active)
				assert.Equal(t, tt.expectedConfig.AllowedTypes, result.AllowedTypes)
				if tt.expectedConfig.Config != nil {
					assert.Equal(t, tt.expectedConfig.Config, result.Config)
				}
			}

			// Verify mocks
			mockClient.AssertExpectations(t)
		})
	}
}

// TestHealthCheck tests the HealthCheck method
func TestHealthCheck(t *testing.T) {
	tests := []healthCheckTestCase{
		{
			name: "Successful Health Check",
			mockClient: func(mc *MockDynamoDBClient) {
				mc.On("DescribeTable", mock.Anything, mock.AnythingOfType("*dynamodb.DescribeTableInput")).Return(&dynamodb.DescribeTableOutput{}, nil)
			},
			expectError: false,
			description: "Should successfully perform health check",
		},
		{
			name: "Health Check Failure",
			mockClient: func(mc *MockDynamoDBClient) {
				mc.On("DescribeTable", mock.Anything, mock.AnythingOfType("*dynamodb.DescribeTableInput")).Return(nil, errors.New("table not found"))
			},
			expectError: true,
			errorMsg:    "table not found",
			description: "Should fail when table doesn't exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockDynamoDBClient{}

			// Setup mocks
			if tt.mockClient != nil {
				tt.mockClient(mockClient)
			}

			// Create repository with mock client
			repo := &DynamoDBRepository{
				client:    mockClient,
				tableName: "test-events",
			}

			// Execute test
			err := repo.HealthCheck(context.Background())

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
			mockClient.AssertExpectations(t)
		})
	}
}

// TestMarshalPayload tests the marshalPayload method
func TestMarshalPayload(t *testing.T) {
	tests := []marshalPayloadTestCase{
		{
			name: "Simple String Values",
			payload: map[string]interface{}{
				"message": "test message",
				"level":   "info",
			},
			expectedResult: map[string]types.AttributeValue{
				"message": &types.AttributeValueMemberS{Value: "test message"},
				"level":   &types.AttributeValueMemberS{Value: "info"},
			},
			description: "Should correctly marshal string values",
		},
		{
			name: "Numeric Values",
			payload: map[string]interface{}{
				"count":    42,
				"amount":   100.50,
				"duration": int64(3600),
			},
			expectedResult: map[string]types.AttributeValue{
				"count":    &types.AttributeValueMemberN{Value: "42"},
				"amount":   &types.AttributeValueMemberN{Value: "100.5"},
				"duration": &types.AttributeValueMemberN{Value: "3600"},
			},
			description: "Should correctly marshal numeric values",
		},
		{
			name: "Boolean Values",
			payload: map[string]interface{}{
				"enabled": true,
				"active":  false,
			},
			expectedResult: map[string]types.AttributeValue{
				"enabled": &types.AttributeValueMemberBOOL{Value: true},
				"active":  &types.AttributeValueMemberBOOL{Value: false},
			},
			description: "Should correctly marshal boolean values",
		},
		{
			name: "String Slice Values",
			payload: map[string]interface{}{
				"tags": []string{"tag1", "tag2", "tag3"},
			},
			expectedResult: map[string]types.AttributeValue{
				"tags": &types.AttributeValueMemberSS{Value: []string{"tag1", "tag2", "tag3"}},
			},
			description: "Should correctly marshal string slice values",
		},
		{
			name: "Nested Map Values",
			payload: map[string]interface{}{
				"metadata": map[string]interface{}{
					"source":  "api",
					"version": "1.0",
				},
			},
			expectedResult: map[string]types.AttributeValue{
				"metadata": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
					"source":  &types.AttributeValueMemberS{Value: "api"},
					"version": &types.AttributeValueMemberS{Value: "1.0"},
				}},
			},
			description: "Should correctly marshal nested map values",
		},
		{
			name: "Complex Nested Structure",
			payload: map[string]interface{}{
				"user": map[string]interface{}{
					"id":     123,
					"name":   "John Doe",
					"active": true,
					"roles":  []string{"admin", "user"},
					"settings": map[string]interface{}{
						"theme": "dark",
						"lang":  "en",
					},
				},
			},
			expectedResult: map[string]types.AttributeValue{
				"user": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
					"id":     &types.AttributeValueMemberN{Value: "123"},
					"name":   &types.AttributeValueMemberS{Value: "John Doe"},
					"active": &types.AttributeValueMemberBOOL{Value: true},
					"roles":  &types.AttributeValueMemberSS{Value: []string{"admin", "user"}},
					"settings": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
						"theme": &types.AttributeValueMemberS{Value: "dark"},
						"lang":  &types.AttributeValueMemberS{Value: "en"},
					}},
				}},
			},
			description: "Should correctly marshal complex nested structures",
		},
		{
			name: "Unknown Type Conversion",
			payload: map[string]interface{}{
				"custom": struct{ value string }{"test"},
			},
			expectedResult: map[string]types.AttributeValue{
				"custom": &types.AttributeValueMemberS{Value: "{test}"},
			},
			description: "Should convert unknown types to string",
		},
		{
			name:           "Empty Payload",
			payload:        map[string]interface{}{},
			expectedResult: map[string]types.AttributeValue{},
			description:    "Should handle empty payload gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create repository
			repo := &DynamoDBRepository{}

			// Execute test
			result := repo.marshalPayload(tt.payload)

			// Assertions
			assert.Equal(t, len(tt.expectedResult), len(result))

			for key, expectedValue := range tt.expectedResult {
				assert.Contains(t, result, key)
				actualValue := result[key]

				// Type-specific assertions
				switch expected := expectedValue.(type) {
				case *types.AttributeValueMemberS:
					if actual, ok := actualValue.(*types.AttributeValueMemberS); ok {
						assert.Equal(t, expected.Value, actual.Value)
					} else {
						t.Errorf("Expected string value for key %s, got %T", key, actualValue)
					}
				case *types.AttributeValueMemberN:
					if actual, ok := actualValue.(*types.AttributeValueMemberN); ok {
						assert.Equal(t, expected.Value, actual.Value)
					} else {
						t.Errorf("Expected numeric value for key %s, got %T", key, actualValue)
					}
				case *types.AttributeValueMemberBOOL:
					if actual, ok := actualValue.(*types.AttributeValueMemberBOOL); ok {
						assert.Equal(t, expected.Value, actual.Value)
					} else {
						t.Errorf("Expected boolean value for key %s, got %T", key, actualValue)
					}
				case *types.AttributeValueMemberSS:
					if actual, ok := actualValue.(*types.AttributeValueMemberSS); ok {
						assert.Equal(t, expected.Value, actual.Value)
					} else {
						t.Errorf("Expected string set value for key %s, got %T", key, actualValue)
					}
				case *types.AttributeValueMemberM:
					if actual, ok := actualValue.(*types.AttributeValueMemberM); ok {
						// For nested maps, we'll just check that they exist
						assert.NotNil(t, actual.Value)
					} else {
						t.Errorf("Expected map value for key %s, got %T", key, actualValue)
					}
				}
			}
		})
	}
}

// TestNewDynamoDBRepository tests the repository constructor
func TestNewDynamoDBRepository(t *testing.T) {
	t.Run("Successful Repository Creation", func(t *testing.T) {
		// Create mock AWS config
		awsCfg := aws.Config{}
		cfg := &config.Config{
			DynamoDBTableName: "test-events",
		}

		// Execute test
		repo := NewDynamoDBRepository(awsCfg, cfg)

		// Assertions
		assert.NotNil(t, repo)
		dynamoRepo, ok := repo.(*DynamoDBRepository)
		assert.True(t, ok)
		assert.Equal(t, "test-events", dynamoRepo.tableName)
		assert.NotNil(t, dynamoRepo.client)
	})
}

// Helper functions to create test data

func createValidProcessedEvent() *models.ProcessedEvent {
	return &models.ProcessedEvent{
		Event: models.Event{
			EventID:   "123e4567-e89b-12d3-a456-426614174000",
			EventType: models.EventTypeMonitoring,
			ClientID:  "client-001",
			Timestamp: time.Now().UTC(),
			Payload: map[string]interface{}{
				"severity": "high",
				"message":  "Test event",
			},
			Version: "1.0",
		},
		ProcessedAt: time.Now().UTC(),
		Status:      models.EventStatusProcessed,
		ErrorMsg:    "",
		RetryCount:  0,
		TTL:         time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
}

func createProcessedEventWithError() *models.ProcessedEvent {
	event := createValidProcessedEvent()
	event.Status = models.EventStatusFailed
	event.ErrorMsg = "Processing failed due to validation error"
	return event
}

func createProcessedEventWithComplexPayload() *models.ProcessedEvent {
	event := createValidProcessedEvent()
	event.Payload = map[string]interface{}{
		"user": map[string]interface{}{
			"id":     123,
			"name":   "John Doe",
			"active": true,
			"roles":  []string{"admin", "user"},
			"settings": map[string]interface{}{
				"theme": "dark",
				"lang":  "en",
			},
		},
		"metadata": map[string]interface{}{
			"source":  "api",
			"version": "1.0",
			"tags":    []string{"important", "urgent"},
		},
	}
	return event
}

func createProcessedEventWithZeroTimestamp() *models.ProcessedEvent {
	event := createValidProcessedEvent()
	event.Timestamp = time.Time{}
	event.ProcessedAt = time.Time{}
	return event
}

func createProcessedEventWithEmptyPayload() *models.ProcessedEvent {
	event := createValidProcessedEvent()
	event.Payload = map[string]interface{}{}
	return event
}
