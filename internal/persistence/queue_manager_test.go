package persistence

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSQSClient is a mock implementation of the SQS client
type MockSQSClient struct {
	mock.Mock
}

func (m *MockSQSClient) ListQueues(ctx context.Context, params *sqs.ListQueuesInput, optFns ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.ListQueuesOutput), args.Error(1)
}

func (m *MockSQSClient) CreateQueue(ctx context.Context, params *sqs.CreateQueueInput, optFns ...func(*sqs.Options)) (*sqs.CreateQueueOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.CreateQueueOutput), args.Error(1)
}

func (m *MockSQSClient) GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.GetQueueUrlOutput), args.Error(1)
}

// Test data structures
type createNewLocalQueuesTestCase struct {
	name           string
	existingQueues []string
	mockClient     func(*MockSQSClient)
	expectError    bool
	errorMsg       string
	description    string
}

type createEventQueueTestCase struct {
	name        string
	mockClient  func(*MockSQSClient)
	expectError bool
	errorMsg    string
	description string
}

type createEventDLQTestCase struct {
	name        string
	mockClient  func(*MockSQSClient)
	expectError bool
	errorMsg    string
	description string
}

type getQueueURLsTestCase struct {
	name         string
	mockClient   func(*MockSQSClient)
	expectError  bool
	errorMsg     string
	expectedURLs map[string]string
	description  string
}

type defaultQueueNamesTestCase struct {
	name          string
	expectedNames *QueueNames
	description   string
}

// TestDefaultQueueNames tests the DefaultQueueNames function
func TestDefaultQueueNames(t *testing.T) {
	tests := []defaultQueueNamesTestCase{
		{
			name: "Default Queue Names",
			expectedNames: &QueueNames{
				EventQueue: "event-queue",
				EventDLQ:   "event-dlq",
			},
			description: "Should return correct default queue names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute test
			result := DefaultQueueNames()

			// Assertions
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedNames.EventQueue, result.EventQueue)
			assert.Equal(t, tt.expectedNames.EventDLQ, result.EventDLQ)
		})
	}
}

// TestNewQueueManager tests the constructor
func TestNewQueueManager(t *testing.T) {
	t.Run("Successful Queue Manager Creation", func(t *testing.T) {
		// Create mock AWS config
		awsCfg := aws.Config{}
		queueNames := DefaultQueueNames()
		logger := logrus.New()

		// Execute test
		manager := NewQueueManager(awsCfg, queueNames, logger)

		// Assertions
		assert.NotNil(t, manager)
		assert.Equal(t, queueNames, manager.queueNames)
		assert.Equal(t, logger, manager.logger)
		assert.NotNil(t, manager.client)
	})
}

// TestCreateNewLocalQueues tests the CreateNewLocalQueues method
func TestCreateNewLocalQueues(t *testing.T) {
	tests := []createNewLocalQueuesTestCase{
		{
			name:           "Successful Queue Creation - No Existing Queues",
			existingQueues: []string{},
			mockClient: func(mc *MockSQSClient) {
				// Mock ListQueues call
				mc.On("ListQueues", mock.Anything, mock.AnythingOfType("*sqs.ListQueuesInput")).Return(&sqs.ListQueuesOutput{
					QueueUrls: []string{},
				}, nil)
				// Mock CreateQueue calls for both queues
				mc.On("CreateQueue", mock.Anything, mock.AnythingOfType("*sqs.CreateQueueInput")).Return(&sqs.CreateQueueOutput{}, nil).Times(2)
			},
			expectError: false,
			description: "Should successfully create both queues when none exist",
		},
		{
			name:           "Successful Queue Creation - Some Existing Queues",
			existingQueues: []string{"other-queue"},
			mockClient: func(mc *MockSQSClient) {
				// Mock ListQueues call
				mc.On("ListQueues", mock.Anything, mock.AnythingOfType("*sqs.ListQueuesInput")).Return(&sqs.ListQueuesOutput{
					QueueUrls: []string{"other-queue"},
				}, nil)
				// Mock CreateQueue calls for both queues
				mc.On("CreateQueue", mock.Anything, mock.AnythingOfType("*sqs.CreateQueueInput")).Return(&sqs.CreateQueueOutput{}, nil).Times(2)
			},
			expectError: false,
			description: "Should successfully create both queues when other queues exist",
		},
		{
			name:           "ListQueues Failure",
			existingQueues: []string{},
			mockClient: func(mc *MockSQSClient) {
				mc.On("ListQueues", mock.Anything, mock.AnythingOfType("*sqs.ListQueuesInput")).Return(nil, errors.New("list queues error"))
			},
			expectError: true,
			errorMsg:    "failed to list queues",
			description: "Should fail when ListQueues operation fails",
		},
		{
			name:           "Event Queue Creation Failure",
			existingQueues: []string{},
			mockClient: func(mc *MockSQSClient) {
				// Mock ListQueues call
				mc.On("ListQueues", mock.Anything, mock.AnythingOfType("*sqs.ListQueuesInput")).Return(&sqs.ListQueuesOutput{
					QueueUrls: []string{},
				}, nil)
				// Mock CreateQueue call for event queue fails
				mc.On("CreateQueue", mock.Anything, mock.AnythingOfType("*sqs.CreateQueueInput")).Return(nil, errors.New("create event queue error"))
			},
			expectError: true,
			errorMsg:    "failed to create event queue",
			description: "Should fail when event queue creation fails",
		},
		{
			name:           "Event DLQ Creation Failure",
			existingQueues: []string{},
			mockClient: func(mc *MockSQSClient) {
				// Mock ListQueues call
				mc.On("ListQueues", mock.Anything, mock.AnythingOfType("*sqs.ListQueuesInput")).Return(&sqs.ListQueuesOutput{
					QueueUrls: []string{},
				}, nil)
				// Mock CreateQueue call for event queue succeeds, but event DLQ fails
				mc.On("CreateQueue", mock.Anything, mock.AnythingOfType("*sqs.CreateQueueInput")).Return(&sqs.CreateQueueOutput{}, nil).Once()
				mc.On("CreateQueue", mock.Anything, mock.AnythingOfType("*sqs.CreateQueueInput")).Return(nil, errors.New("create event DLQ error")).Once()
			},
			expectError: true,
			errorMsg:    "failed to create event DLQ",
			description: "Should fail when event DLQ creation fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockClient := &MockSQSClient{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockClient != nil {
				tt.mockClient(mockClient)
			}

			// Create queue manager with mock client
			manager := &QueueManager{
				client:     mockClient,
				queueNames: DefaultQueueNames(),
				logger:     logger,
			}

			// Execute test
			err := manager.CreateNewLocalQueues(context.Background())

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

// TestCreateEventQueue tests the createEventQueue method
func TestCreateEventQueue(t *testing.T) {
	tests := []createEventQueueTestCase{
		{
			name: "Successful Event Queue Creation",
			mockClient: func(mc *MockSQSClient) {
				mc.On("CreateQueue", mock.Anything, mock.AnythingOfType("*sqs.CreateQueueInput")).Return(&sqs.CreateQueueOutput{}, nil)
			},
			expectError: false,
			description: "Should successfully create event queue with correct attributes",
		},
		{
			name: "Event Queue Creation Failure",
			mockClient: func(mc *MockSQSClient) {
				mc.On("CreateQueue", mock.Anything, mock.AnythingOfType("*sqs.CreateQueueInput")).Return(nil, errors.New("create queue error"))
			},
			expectError: true,
			errorMsg:    "unable to create event queue",
			description: "Should fail when CreateQueue operation fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockClient := &MockSQSClient{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockClient != nil {
				tt.mockClient(mockClient)
			}

			// Create queue manager with mock client
			manager := &QueueManager{
				client:     mockClient,
				queueNames: DefaultQueueNames(),
				logger:     logger,
			}

			// Execute test
			err := manager.createEventQueue(context.Background())

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

// TestCreateEventDLQ tests the createEventDLQ method
func TestCreateEventDLQ(t *testing.T) {
	tests := []createEventDLQTestCase{
		{
			name: "Successful Event DLQ Creation",
			mockClient: func(mc *MockSQSClient) {
				mc.On("CreateQueue", mock.Anything, mock.AnythingOfType("*sqs.CreateQueueInput")).Return(&sqs.CreateQueueOutput{}, nil)
			},
			expectError: false,
			description: "Should successfully create event DLQ with correct attributes",
		},
		{
			name: "Event DLQ Creation Failure",
			mockClient: func(mc *MockSQSClient) {
				mc.On("CreateQueue", mock.Anything, mock.AnythingOfType("*sqs.CreateQueueInput")).Return(nil, errors.New("create queue error"))
			},
			expectError: true,
			errorMsg:    "unable to create event DLQ",
			description: "Should fail when CreateQueue operation fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockClient := &MockSQSClient{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockClient != nil {
				tt.mockClient(mockClient)
			}

			// Create queue manager with mock client
			manager := &QueueManager{
				client:     mockClient,
				queueNames: DefaultQueueNames(),
				logger:     logger,
			}

			// Execute test
			err := manager.createEventDLQ(context.Background())

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

// TestGetQueueURLs tests the GetQueueURLs method
func TestGetQueueURLs(t *testing.T) {
	tests := []getQueueURLsTestCase{
		{
			name: "Successful Queue URLs Retrieval",
			mockClient: func(mc *MockSQSClient) {
				// Mock GetQueueUrl calls for both queues
				mc.On("GetQueueUrl", mock.Anything, mock.AnythingOfType("*sqs.GetQueueUrlInput")).Return(&sqs.GetQueueUrlOutput{
					QueueUrl: aws.String("https://sqs.test.com/event-queue"),
				}, nil).Once()
				mc.On("GetQueueUrl", mock.Anything, mock.AnythingOfType("*sqs.GetQueueUrlInput")).Return(&sqs.GetQueueUrlOutput{
					QueueUrl: aws.String("https://sqs.test.com/event-dlq"),
				}, nil).Once()
			},
			expectError: false,
			expectedURLs: map[string]string{
				"event_queue": "https://sqs.test.com/event-queue",
				"event_dlq":   "https://sqs.test.com/event-dlq",
			},
			description: "Should successfully retrieve URLs for both queues",
		},
		{
			name: "Event Queue URL Retrieval Failure",
			mockClient: func(mc *MockSQSClient) {
				// Mock GetQueueUrl call for event queue fails
				mc.On("GetQueueUrl", mock.Anything, mock.AnythingOfType("*sqs.GetQueueUrlInput")).Return(nil, errors.New("get queue URL error"))
			},
			expectError: true,
			errorMsg:    "failed to get event queue URL",
			description: "Should fail when event queue URL retrieval fails",
		},
		{
			name: "Event DLQ URL Retrieval Failure",
			mockClient: func(mc *MockSQSClient) {
				// Mock GetQueueUrl call for event queue succeeds, but event DLQ fails
				mc.On("GetQueueUrl", mock.Anything, mock.AnythingOfType("*sqs.GetQueueUrlInput")).Return(&sqs.GetQueueUrlOutput{
					QueueUrl: aws.String("https://sqs.test.com/event-queue"),
				}, nil).Once()
				mc.On("GetQueueUrl", mock.Anything, mock.AnythingOfType("*sqs.GetQueueUrlInput")).Return(nil, errors.New("get queue URL error")).Once()
			},
			expectError: true,
			errorMsg:    "failed to get event DLQ URL",
			description: "Should fail when event DLQ URL retrieval fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockClient := &MockSQSClient{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockClient != nil {
				tt.mockClient(mockClient)
			}

			// Create queue manager with mock client
			manager := &QueueManager{
				client:     mockClient,
				queueNames: DefaultQueueNames(),
				logger:     logger,
			}

			// Execute test
			result, err := manager.GetQueueURLs(context.Background())

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
				assert.Equal(t, tt.expectedURLs, result)
			}

			// Verify mocks
			mockClient.AssertExpectations(t)
		})
	}
}
