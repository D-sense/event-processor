package persistence

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test data structures
type createNewLocalTablesTestCase struct {
	name           string
	existingTables []string
	mockClient     func(*MockDynamoDBClient)
	expectError    bool
	errorMsg       string
	description    string
}

type createEventsTableTestCase struct {
	name        string
	mockClient  func(*MockDynamoDBClient)
	expectError bool
	errorMsg    string
	description string
}

type createEventsClientsTableTestCase struct {
	name        string
	mockClient  func(*MockDynamoDBClient)
	expectError bool
	errorMsg    string
	description string
}

type insertSampleClientConfigsTestCase struct {
	name        string
	mockClient  func(*MockDynamoDBClient)
	expectError bool
	errorMsg    string
	description string
}

type defaultTableNamesTestCase struct {
	name          string
	expectedNames *TableNames
	description   string
}

// TestDefaultTableNames tests the DefaultTableNames function
func TestDefaultTableNames(t *testing.T) {
	tests := []defaultTableNamesTestCase{
		{
			name: "Default Table Names",
			expectedNames: &TableNames{
				Events:        "events",
				EventsClients: "events-clients",
			},
			description: "Should return correct default table names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute test
			result := DefaultTableNames()

			// Assertions
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedNames.Events, result.Events)
			assert.Equal(t, tt.expectedNames.EventsClients, result.EventsClients)
		})
	}
}

// TestNewTableManager tests the constructor
func TestNewTableManager(t *testing.T) {
	t.Run("Successful Table Manager Creation", func(t *testing.T) {
		// Create mock AWS config
		awsCfg := aws.Config{}
		tableNames := DefaultTableNames()
		logger := logrus.New()

		// Execute test
		manager := NewTableManager(awsCfg, tableNames, logger)

		// Assertions
		assert.NotNil(t, manager)
		assert.Equal(t, tableNames, manager.tableNames)
		assert.Equal(t, logger, manager.logger)
		assert.NotNil(t, manager.client)
	})
}

// TestCreateNewLocalTables tests the CreateNewLocalTables method
func TestCreateNewLocalTables(t *testing.T) {
	tests := []createNewLocalTablesTestCase{
		{
			name:           "Successful Table Creation - No Existing Tables",
			existingTables: []string{},
			mockClient: func(mc *MockDynamoDBClient) {
				// Mock ListTables call
				mc.On("ListTables", mock.Anything, mock.AnythingOfType("*dynamodb.ListTablesInput")).Return(&dynamodb.ListTablesOutput{
					TableNames: []string{},
				}, nil)
				// Mock CreateTable calls for both tables
				mc.On("CreateTable", mock.Anything, mock.AnythingOfType("*dynamodb.CreateTableInput")).Return(&dynamodb.CreateTableOutput{}, nil).Times(2)
			},
			expectError: false,
			description: "Should successfully create both tables when none exist",
		},
		{
			name:           "Successful Table Creation - Some Existing Tables",
			existingTables: []string{"other-table"},
			mockClient: func(mc *MockDynamoDBClient) {
				// Mock ListTables call
				mc.On("ListTables", mock.Anything, mock.AnythingOfType("*dynamodb.ListTablesInput")).Return(&dynamodb.ListTablesOutput{
					TableNames: []string{"other-table"},
				}, nil)
				// Mock CreateTable calls for both tables
				mc.On("CreateTable", mock.Anything, mock.AnythingOfType("*dynamodb.CreateTableInput")).Return(&dynamodb.CreateTableOutput{}, nil).Times(2)
			},
			expectError: false,
			description: "Should successfully create both tables when other tables exist",
		},
		{
			name:           "ListTables Failure",
			existingTables: []string{},
			mockClient: func(mc *MockDynamoDBClient) {
				mc.On("ListTables", mock.Anything, mock.AnythingOfType("*dynamodb.ListTablesInput")).Return(nil, errors.New("list tables error"))
			},
			expectError: true,
			errorMsg:    "failed to list tables",
			description: "Should fail when ListTables operation fails",
		},
		{
			name:           "Events Table Creation Failure",
			existingTables: []string{},
			mockClient: func(mc *MockDynamoDBClient) {
				// Mock ListTables call
				mc.On("ListTables", mock.Anything, mock.AnythingOfType("*dynamodb.ListTablesInput")).Return(&dynamodb.ListTablesOutput{
					TableNames: []string{},
				}, nil)
				// Mock CreateTable call for events table fails
				mc.On("CreateTable", mock.Anything, mock.AnythingOfType("*dynamodb.CreateTableInput")).Return(nil, errors.New("create events table error"))
			},
			expectError: true,
			errorMsg:    "failed to create events table",
			description: "Should fail when events table creation fails",
		},
		{
			name:           "EventsClients Table Creation Failure",
			existingTables: []string{},
			mockClient: func(mc *MockDynamoDBClient) {
				// Mock ListTables call
				mc.On("ListTables", mock.Anything, mock.AnythingOfType("*dynamodb.ListTablesInput")).Return(&dynamodb.ListTablesOutput{
					TableNames: []string{},
				}, nil)
				// Mock CreateTable call for events table succeeds, but events-clients fails
				mc.On("CreateTable", mock.Anything, mock.AnythingOfType("*dynamodb.CreateTableInput")).Return(&dynamodb.CreateTableOutput{}, nil).Once()
				mc.On("CreateTable", mock.Anything, mock.AnythingOfType("*dynamodb.CreateTableInput")).Return(nil, errors.New("create events-clients table error")).Once()
			},
			expectError: true,
			errorMsg:    "failed to create events-clients table",
			description: "Should fail when events-clients table creation fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockClient := &MockDynamoDBClient{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockClient != nil {
				tt.mockClient(mockClient)
			}

			// Create table manager with mock client
			manager := &TableManager{
				client:     mockClient,
				tableNames: DefaultTableNames(),
				logger:     logger,
			}

			// Execute test
			err := manager.CreateNewLocalTables(context.Background())

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

// TestCreateEventsTable tests the createEventsTable method
func TestCreateEventsTable(t *testing.T) {
	tests := []createEventsTableTestCase{
		{
			name: "Successful Events Table Creation",
			mockClient: func(mc *MockDynamoDBClient) {
				mc.On("CreateTable", mock.Anything, mock.AnythingOfType("*dynamodb.CreateTableInput")).Return(&dynamodb.CreateTableOutput{}, nil)
			},
			expectError: false,
			description: "Should successfully create events table with correct schema",
		},
		{
			name: "Events Table Creation Failure",
			mockClient: func(mc *MockDynamoDBClient) {
				mc.On("CreateTable", mock.Anything, mock.AnythingOfType("*dynamodb.CreateTableInput")).Return(nil, errors.New("create table error"))
			},
			expectError: true,
			errorMsg:    "unable to create 'events' DynamoDB table",
			description: "Should fail when CreateTable operation fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockClient := &MockDynamoDBClient{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockClient != nil {
				tt.mockClient(mockClient)
			}

			// Create table manager with mock client
			manager := &TableManager{
				client:     mockClient,
				tableNames: DefaultTableNames(),
				logger:     logger,
			}

			// Execute test
			err := manager.createEventsTable(context.Background())

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

// TestCreateEventsClientsTable tests the createEventsClientsTable method
func TestCreateEventsClientsTable(t *testing.T) {
	tests := []createEventsClientsTableTestCase{
		{
			name: "Successful EventsClients Table Creation",
			mockClient: func(mc *MockDynamoDBClient) {
				mc.On("CreateTable", mock.Anything, mock.AnythingOfType("*dynamodb.CreateTableInput")).Return(&dynamodb.CreateTableOutput{}, nil)
			},
			expectError: false,
			description: "Should successfully create events-clients table with correct schema",
		},
		{
			name: "EventsClients Table Creation Failure",
			mockClient: func(mc *MockDynamoDBClient) {
				mc.On("CreateTable", mock.Anything, mock.AnythingOfType("*dynamodb.CreateTableInput")).Return(nil, errors.New("create table error"))
			},
			expectError: true,
			errorMsg:    "unable to create 'events-clients' DynamoDB table",
			description: "Should fail when CreateTable operation fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockClient := &MockDynamoDBClient{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockClient != nil {
				tt.mockClient(mockClient)
			}

			// Create table manager with mock client
			manager := &TableManager{
				client:     mockClient,
				tableNames: DefaultTableNames(),
				logger:     logger,
			}

			// Execute test
			err := manager.createEventsClientsTable(context.Background())

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

// TestInsertSampleClientConfigs tests the InsertSampleClientConfigs method
func TestInsertSampleClientConfigs(t *testing.T) {
	tests := []insertSampleClientConfigsTestCase{
		{
			name: "Successful Sample Client Configs Insertion",
			mockClient: func(mc *MockDynamoDBClient) {
				// Mock PutItem calls for all three client configs
				mc.On("PutItem", mock.Anything, mock.AnythingOfType("*dynamodb.PutItemInput")).Return(&dynamodb.PutItemOutput{}, nil).Times(3)
			},
			expectError: false,
			description: "Should successfully insert all sample client configurations",
		},
		{
			name: "First Client Config Insertion Failure",
			mockClient: func(mc *MockDynamoDBClient) {
				// Mock PutItem call for first client config fails
				mc.On("PutItem", mock.Anything, mock.AnythingOfType("*dynamodb.PutItemInput")).Return(nil, errors.New("put item error"))
			},
			expectError: true,
			errorMsg:    "failed to insert client config",
			description: "Should fail when first client config insertion fails",
		},
		{
			name: "Second Client Config Insertion Failure",
			mockClient: func(mc *MockDynamoDBClient) {
				// Mock PutItem call for first client config succeeds, second fails
				mc.On("PutItem", mock.Anything, mock.AnythingOfType("*dynamodb.PutItemInput")).Return(&dynamodb.PutItemOutput{}, nil).Once()
				mc.On("PutItem", mock.Anything, mock.AnythingOfType("*dynamodb.PutItemInput")).Return(nil, errors.New("put item error")).Once()
			},
			expectError: true,
			errorMsg:    "failed to insert client config",
			description: "Should fail when second client config insertion fails",
		},
		{
			name: "Third Client Config Insertion Failure",
			mockClient: func(mc *MockDynamoDBClient) {
				// Mock PutItem calls for first two client configs succeed, third fails
				mc.On("PutItem", mock.Anything, mock.AnythingOfType("*dynamodb.PutItemInput")).Return(&dynamodb.PutItemOutput{}, nil).Times(2)
				mc.On("PutItem", mock.Anything, mock.AnythingOfType("*dynamodb.PutItemInput")).Return(nil, errors.New("put item error")).Once()
			},
			expectError: true,
			errorMsg:    "failed to insert client config",
			description: "Should fail when third client config insertion fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockClient := &MockDynamoDBClient{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockClient != nil {
				tt.mockClient(mockClient)
			}

			// Create table manager with mock client
			manager := &TableManager{
				client:     mockClient,
				tableNames: DefaultTableNames(),
				logger:     logger,
			}

			// Execute test
			err := manager.InsertSampleClientConfigs(context.Background())

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
