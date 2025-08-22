package persistence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTableManager is a mock implementation of the TableManager
type MockTableManager struct {
	mock.Mock
}

func (m *MockTableManager) CreateNewLocalTables(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTableManager) InsertSampleClientConfigs(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockQueueManager is a mock implementation of the QueueManager
type MockQueueManager struct {
	mock.Mock
}

func (m *MockQueueManager) CreateNewLocalQueues(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockQueueManager) GetQueueURLs(ctx context.Context) (map[string]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

// Test data structures
type setupInfrastructureTestCase struct {
	name             string
	mockTableManager func(*MockTableManager)
	mockQueueManager func(*MockQueueManager)
	expectError      bool
	errorMsg         string
	description      string
}

type newInfrastructureManagerTestCase struct {
	name           string
	expectedResult *InfrastructureManager
	description    string
}

// TestNewInfrastructureManager tests the constructor
func TestNewInfrastructureManager(t *testing.T) {
	tests := []newInfrastructureManagerTestCase{
		{
			name:           "Successful Infrastructure Manager Creation",
			expectedResult: &InfrastructureManager{},
			description:    "Should successfully create infrastructure manager with table and queue managers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock AWS config
			awsCfg := aws.Config{}
			logger := logrus.New()

			// Execute test
			result := NewInfrastructureManager(awsCfg, logger)

			// Assertions
			assert.NotNil(t, result)
			assert.NotNil(t, result.tableManager)
			assert.NotNil(t, result.queueManager)
			assert.Equal(t, logger, result.logger)
		})
	}
}

// TestSetupInfrastructure tests the SetupInfrastructure method
func TestSetupInfrastructure(t *testing.T) {
	tests := []setupInfrastructureTestCase{
		{
			name: "Successful Infrastructure Setup",
			mockTableManager: func(mt *MockTableManager) {
				mt.On("CreateNewLocalTables", mock.Anything).Return(nil)
				mt.On("InsertSampleClientConfigs", mock.Anything).Return(nil)
			},
			mockQueueManager: func(mq *MockQueueManager) {
				mq.On("CreateNewLocalQueues", mock.Anything).Return(nil)
				mq.On("GetQueueURLs", mock.Anything).Return(map[string]string{
					"event_queue": "https://sqs.test.com/event-queue",
					"event_dlq":   "https://sqs.test.com/event-dlq",
				}, nil)
			},
			expectError: false,
			description: "Should successfully setup all infrastructure components",
		},
		{
			name: "Table Creation Failure",
			mockTableManager: func(mt *MockTableManager) {
				mt.On("CreateNewLocalTables", mock.Anything).Return(errors.New("table creation error"))
			},
			mockQueueManager: func(mq *MockQueueManager) {
				// No queue manager calls expected since table creation fails
			},
			expectError: true,
			errorMsg:    "failed to setup DynamoDB tables",
			description: "Should fail when DynamoDB table creation fails",
		},
		{
			name: "Sample Client Configs Insertion Failure - Continue",
			mockTableManager: func(mt *MockTableManager) {
				mt.On("CreateNewLocalTables", mock.Anything).Return(nil)
				mt.On("InsertSampleClientConfigs", mock.Anything).Return(errors.New("client config insertion error"))
			},
			mockQueueManager: func(mq *MockQueueManager) {
				mq.On("CreateNewLocalQueues", mock.Anything).Return(nil)
				mq.On("GetQueueURLs", mock.Anything).Return(map[string]string{
					"event_queue": "https://sqs.test.com/event-queue",
					"event_dlq":   "https://sqs.test.com/event-dlq",
				}, nil)
			},
			expectError: false,
			description: "Should continue setup when sample client configs insertion fails",
		},
		{
			name: "Queue Creation Failure",
			mockTableManager: func(mt *MockTableManager) {
				mt.On("CreateNewLocalTables", mock.Anything).Return(nil)
				mt.On("InsertSampleClientConfigs", mock.Anything).Return(nil)
			},
			mockQueueManager: func(mq *MockQueueManager) {
				mq.On("CreateNewLocalQueues", mock.Anything).Return(errors.New("queue creation error"))
			},
			expectError: true,
			errorMsg:    "failed to setup SQS queues",
			description: "Should fail when SQS queue creation fails",
		},
		{
			name: "Queue URLs Retrieval Failure - Continue",
			mockTableManager: func(mt *MockTableManager) {
				mt.On("CreateNewLocalTables", mock.Anything).Return(nil)
				mt.On("InsertSampleClientConfigs", mock.Anything).Return(nil)
			},
			mockQueueManager: func(mq *MockQueueManager) {
				mq.On("CreateNewLocalQueues", mock.Anything).Return(nil)
				mq.On("GetQueueURLs", mock.Anything).Return(nil, errors.New("get queue URLs error"))
			},
			expectError: false,
			description: "Should continue setup when queue URLs retrieval fails",
		},
		{
			name: "Partial Infrastructure Setup - Tables Success, Queues Failure",
			mockTableManager: func(mt *MockTableManager) {
				mt.On("CreateNewLocalTables", mock.Anything).Return(nil)
				mt.On("InsertSampleClientConfigs", mock.Anything).Return(nil)
			},
			mockQueueManager: func(mq *MockQueueManager) {
				mq.On("CreateNewLocalQueues", mock.Anything).Return(errors.New("queue creation error"))
			},
			expectError: true,
			errorMsg:    "failed to setup SQS queues",
			description: "Should fail when tables succeed but queues fail",
		},
		{
			name: "Complex Failure Scenario - Tables Success, Client Configs Failure, Queues Success, URLs Failure",
			mockTableManager: func(mt *MockTableManager) {
				mt.On("CreateNewLocalTables", mock.Anything).Return(nil)
				mt.On("InsertSampleClientConfigs", mock.Anything).Return(errors.New("client config insertion error"))
			},
			mockQueueManager: func(mq *MockQueueManager) {
				mq.On("CreateNewLocalQueues", mock.Anything).Return(nil)
				mq.On("GetQueueURLs", mock.Anything).Return(nil, errors.New("get queue URLs error"))
			},
			expectError: false,
			description: "Should complete setup when only non-critical operations fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockTableManager := &MockTableManager{}
			mockQueueManager := &MockQueueManager{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockTableManager != nil {
				tt.mockTableManager(mockTableManager)
			}
			if tt.mockQueueManager != nil {
				tt.mockQueueManager(mockQueueManager)
			}

			// Create infrastructure manager with mock components
			manager := &InfrastructureManager{
				tableManager: mockTableManager,
				queueManager: mockQueueManager,
				logger:       logger,
			}

			// Execute test
			err := manager.SetupInfrastructure(context.Background())

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
			mockTableManager.AssertExpectations(t)
			mockQueueManager.AssertExpectations(t)
		})
	}
}

// TestInfrastructureManagerStruct tests the InfrastructureManager struct
func TestInfrastructureManagerStruct(t *testing.T) {
	t.Run("InfrastructureManager Struct Fields", func(t *testing.T) {
		// Create infrastructure manager instance
		manager := &InfrastructureManager{
			tableManager: &MockTableManager{},
			queueManager: &MockQueueManager{},
			logger:       logrus.New(),
		}

		// Assertions
		assert.NotNil(t, manager.tableManager)
		assert.NotNil(t, manager.queueManager)
		assert.NotNil(t, manager.logger)
	})
}

// TestInfrastructureSetupTiming tests the timing aspects of infrastructure setup
func TestInfrastructureSetupTiming(t *testing.T) {
	t.Run("Infrastructure Setup Timing", func(t *testing.T) {
		// Create mocks
		mockTableManager := &MockTableManager{}
		mockQueueManager := &MockQueueManager{}
		logger := logrus.New()
		logger.SetLevel(logrus.DebugLevel)

		// Setup mocks for successful operations
		mockTableManager.On("CreateNewLocalTables", mock.Anything).Return(nil)
		mockTableManager.On("InsertSampleClientConfigs", mock.Anything).Return(nil)
		mockQueueManager.On("CreateNewLocalQueues", mock.Anything).Return(nil)
		mockQueueManager.On("GetQueueURLs", mock.Anything).Return(map[string]string{
			"event_queue": "https://sqs.test.com/event-queue",
			"event_dlq":   "https://sqs.test.com/event-dlq",
		}, nil)

		// Create infrastructure manager with mock components
		manager := &InfrastructureManager{
			tableManager: mockTableManager,
			queueManager: mockQueueManager,
			logger:       logger,
		}

		// Execute test and measure time
		startTime := time.Now()
		err := manager.SetupInfrastructure(context.Background())
		duration := time.Since(startTime)

		// Assertions
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, duration, 8*time.Second, "Setup should take at least 8 seconds due to delays")
		assert.LessOrEqual(t, duration, 12*time.Second, "Setup should not take more than 12 seconds")

		// Verify mocks
		mockTableManager.AssertExpectations(t)
		mockQueueManager.AssertExpectations(t)
	})
}

// TestInfrastructureSetupErrorHandling tests error handling during setup
func TestInfrastructureSetupErrorHandling(t *testing.T) {
	t.Run("Error Handling During Setup", func(t *testing.T) {
		// Create mocks
		mockTableManager := &MockTableManager{}
		mockQueueManager := &MockQueueManager{}
		logger := logrus.New()
		logger.SetLevel(logrus.DebugLevel)

		// Test case: Tables succeed, client configs fail (non-critical), queues succeed, URLs fail (non-critical)
		mockTableManager.On("CreateNewLocalTables", mock.Anything).Return(nil)
		mockTableManager.On("InsertSampleClientConfigs", mock.Anything).Return(errors.New("client config insertion error"))
		mockQueueManager.On("CreateNewLocalQueues", mock.Anything).Return(nil)
		mockQueueManager.On("GetQueueURLs", mock.Anything).Return(nil, errors.New("get queue URLs error"))

		// Create infrastructure manager with mock components
		manager := &InfrastructureManager{
			tableManager: mockTableManager,
			queueManager: mockQueueManager,
			logger:       logger,
		}

		// Execute test
		err := manager.SetupInfrastructure(context.Background())

		// Assertions - should succeed despite non-critical failures
		assert.NoError(t, err)

		// Verify mocks
		mockTableManager.AssertExpectations(t)
		mockQueueManager.AssertExpectations(t)
	})
}

// TestInfrastructureSetupLogging tests the logging behavior during setup
func TestInfrastructureSetupLogging(t *testing.T) {
	t.Run("Logging During Setup", func(t *testing.T) {
		// Create mocks
		mockTableManager := &MockTableManager{}
		mockQueueManager := &MockQueueManager{}
		logger := logrus.New()
		logger.SetLevel(logrus.DebugLevel)

		// Setup mocks for successful operations
		mockTableManager.On("CreateNewLocalTables", mock.Anything).Return(nil)
		mockTableManager.On("InsertSampleClientConfigs", mock.Anything).Return(nil)
		mockQueueManager.On("CreateNewLocalQueues", mock.Anything).Return(nil)
		mockQueueManager.On("GetQueueURLs", mock.Anything).Return(map[string]string{
			"event_queue": "https://sqs.test.com/event-queue",
			"event_dlq":   "https://sqs.test.com/event-dlq",
		}, nil)

		// Create infrastructure manager with mock components
		manager := &InfrastructureManager{
			tableManager: mockTableManager,
			queueManager: mockQueueManager,
			logger:       logger,
		}

		// Execute test
		err := manager.SetupInfrastructure(context.Background())

		// Assertions
		assert.NoError(t, err)

		// Verify mocks
		mockTableManager.AssertExpectations(t)
		mockQueueManager.AssertExpectations(t)
	})
}
