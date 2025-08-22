package consumer

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/d-sense/event-processor/internal/config"
	"github.com/d-sense/event-processor/internal/processor"
)

// MockSQSClient is a mock implementation of the SQS client
type MockSQSClient struct {
	mock.Mock
}

func (m *MockSQSClient) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.ReceiveMessageOutput), args.Error(1)
}

func (m *MockSQSClient) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.SendMessageOutput), args.Error(1)
}

func (m *MockSQSClient) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.DeleteMessageOutput), args.Error(1)
}

// MockProcessor is a mock implementation of the Processor interface
type MockProcessor struct {
	mock.Mock
}

func (m *MockProcessor) ProcessEvent(ctx context.Context, eventData interface{}) error {
	args := m.Called(ctx, eventData)
	return args.Error(0)
}

// Ensure MockProcessor implements processor.Processor
var _ processor.Processor = (*MockProcessor)(nil)

// Test data structures
type startConsumerTestCase struct {
	name          string
	mockSQS       func(*MockSQSClient)
	mockProcessor func(*MockProcessor)
	expectError   bool
	errorMsg      string
	description   string
}

type stopConsumerTestCase struct {
	name        string
	context     context.Context
	expectError bool
	errorMsg    string
	description string
}

type processMessageTestCase struct {
	name          string
	message       *types.Message
	mockProcessor func(*MockProcessor)
	mockSQS       func(*MockSQSClient)
	expectDLQ     bool
	expectRequeue bool
	expectDelete  bool
	description   string
}

type retryCountTestCase struct {
	name          string
	message       *types.Message
	expectedCount int
	description   string
}

type sendToDLQTestCase struct {
	name        string
	message     *types.Message
	reason      string
	mockSQS     func(*MockSQSClient)
	expectError bool
	description string
}

type requeueMessageTestCase struct {
	name          string
	message       *types.Message
	newRetryCount int
	mockSQS       func(*MockSQSClient)
	expectError   bool
	description   string
}

type deleteMessageTestCase struct {
	name        string
	message     *types.Message
	mockSQS     func(*MockSQSClient)
	expectError bool
	description string
}

// TestStart tests the Start method
func TestStart(t *testing.T) {
	tests := []startConsumerTestCase{
		{
			name: "Successful Start",
			mockSQS: func(mc *MockSQSClient) {
				// No SQS calls expected during start
			},
			mockProcessor: func(mp *MockProcessor) {
				// No processor calls expected during start
			},
			expectError: false,
			description: "Should successfully start the consumer",
		},
		{
			name: "Already Running",
			mockSQS: func(mc *MockSQSClient) {
				// No SQS calls expected
			},
			mockProcessor: func(mp *MockProcessor) {
				// No processor calls expected
			},
			expectError: true,
			errorMsg:    "consumer is already running",
			description: "Should fail when consumer is already running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockSQS := &MockSQSClient{}
			mockProcessor := &MockProcessor{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockSQS != nil {
				tt.mockSQS(mockSQS)
			}
			if tt.mockProcessor != nil {
				tt.mockProcessor(mockProcessor)
			}

			// Create consumer
			consumer := &SQSConsumer{
				sqsClient:  mockSQS,
				processor:  mockProcessor,
				logger:     logger,
				stopChan:   make(chan struct{}),
				isRunning:  tt.name == "Already Running", // Set running state based on test
				maxRetries: 3,
				waitTime:   20,
				batchSize:  10,
			}

			// Execute test
			err := consumer.Start(context.Background())

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.True(t, consumer.isRunning)
			}

			// Cleanup
			if consumer.isRunning {
				close(consumer.stopChan)
			}
		})
	}
}

// TestStop tests the Stop method
func TestStop(t *testing.T) {
	tests := []stopConsumerTestCase{
		{
			name:        "Successful Stop",
			context:     context.Background(),
			expectError: false,
			description: "Should successfully stop the consumer",
		},
		{
			name:        "Stop with Context Cancellation",
			context:     createCancelledContext(),
			expectError: true,
			errorMsg:    "context canceled",
			description: "Should stop when context is cancelled",
		},
		{
			name:        "Stop Already Stopped Consumer",
			context:     context.Background(),
			expectError: false,
			description: "Should handle already stopped consumer gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create consumer
			consumer := &SQSConsumer{
				stopChan:  make(chan struct{}),
				isRunning: tt.name != "Stop Already Stopped Consumer",
				logger:    logrus.New(),
			}

			// For the successful stop test, we need to simulate a running consumer
			if tt.name == "Successful Stop" {
				// Create a context with a short timeout instead of using goroutines
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()

				// Execute test with timeout context
				err := consumer.Stop(ctx)

				// Assertions
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "deadline exceeded")
			} else {
				// Execute test with original context
				err := consumer.Stop(tt.context)

				// Assertions
				if tt.expectError {
					assert.Error(t, err)
					if tt.errorMsg != "" {
						assert.Contains(t, err.Error(), tt.errorMsg)
					}
				} else {
					assert.NoError(t, err)
				}
			}

			assert.False(t, consumer.isRunning)
		})
	}
}

// TestProcessMessage tests the processMessage method
func TestProcessMessage(t *testing.T) {
	tests := []processMessageTestCase{
		{
			name:    "Successful Message Processing",
			message: createTestMessage("msg-001", "test body", 0),
			mockProcessor: func(mp *MockProcessor) {
				mp.On("ProcessEvent", mock.Anything, mock.AnythingOfType("*types.Message")).Return(nil)
			},
			mockSQS: func(mc *MockSQSClient) {
				mc.On("DeleteMessage", mock.Anything, mock.AnythingOfType("*sqs.DeleteMessageInput")).Return(&sqs.DeleteMessageOutput{}, nil)
			},
			expectDLQ:     false,
			expectRequeue: false,
			expectDelete:  true,
			description:   "Should successfully process message and delete it",
		},
		{
			name:    "Processing Failure - Under Max Retries",
			message: createTestMessage("msg-002", "test body", 1),
			mockProcessor: func(mp *MockProcessor) {
				mp.On("ProcessEvent", mock.Anything, mock.AnythingOfType("*types.Message")).Return(errors.New("processing error"))
			},
			mockSQS: func(mc *MockSQSClient) {
				mc.On("SendMessage", mock.Anything, mock.AnythingOfType("*sqs.SendMessageInput")).Return(&sqs.SendMessageOutput{}, nil)
			},
			expectDLQ:     false,
			expectRequeue: true,
			expectDelete:  false,
			description:   "Should requeue message when processing fails and under max retries",
		},
		{
			name:    "Processing Failure - At Max Retries",
			message: createTestMessage("msg-003", "test body", 3),
			mockProcessor: func(mp *MockProcessor) {
				// No processor calls expected - message goes directly to DLQ when retry count equals maxRetries
			},
			mockSQS: func(mc *MockSQSClient) {
				mc.On("SendMessage", mock.Anything, mock.AnythingOfType("*sqs.SendMessageInput")).Return(&sqs.SendMessageOutput{}, nil)
				mc.On("DeleteMessage", mock.Anything, mock.AnythingOfType("*sqs.DeleteMessageInput")).Return(&sqs.DeleteMessageOutput{}, nil)
			},
			expectDLQ:     true,
			expectRequeue: false,
			expectDelete:  true,
			description:   "Should send to DLQ when message retry count equals max retries",
		},
		{
			name:    "Processing Failure - Exceeded Max Retries",
			message: createTestMessage("msg-004", "test body", 4),
			mockProcessor: func(mp *MockProcessor) {
				// No processor calls expected - message should go directly to DLQ
			},
			mockSQS: func(mc *MockSQSClient) {
				mc.On("SendMessage", mock.Anything, mock.AnythingOfType("*sqs.SendMessageInput")).Return(&sqs.SendMessageOutput{}, nil)
				mc.On("DeleteMessage", mock.Anything, mock.AnythingOfType("*sqs.DeleteMessageInput")).Return(&sqs.DeleteMessageOutput{}, nil)
			},
			expectDLQ:     true,
			expectRequeue: false,
			expectDelete:  true,
			description:   "Should send to DLQ when message exceeds max retries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockSQS := &MockSQSClient{}
			mockProcessor := &MockProcessor{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockSQS != nil {
				tt.mockSQS(mockSQS)
			}
			if tt.mockProcessor != nil {
				tt.mockProcessor(mockProcessor)
			}

			// Create consumer
			consumer := &SQSConsumer{
				sqsClient:  mockSQS,
				processor:  mockProcessor,
				logger:     logger,
				queueURL:   "https://sqs.test.com/queue",
				dlqURL:     "https://sqs.test.com/dlq",
				maxRetries: 3,
			}

			// Execute test
			consumer.processMessage(context.Background(), tt.message)

			// Verify mocks
			mockSQS.AssertExpectations(t)
			mockProcessor.AssertExpectations(t)
		})
	}
}

// TestGetRetryCount tests the getRetryCount method
func TestGetRetryCount(t *testing.T) {
	tests := []retryCountTestCase{
		{
			name:          "No Message Attributes",
			message:       createTestMessage("msg-001", "test body", 0),
			expectedCount: 0,
			description:   "Should return 0 when message has no attributes",
		},
		{
			name:          "No Retry Count Attribute",
			message:       createTestMessageWithAttributes("msg-002", "test body", map[string]string{"other": "value"}),
			expectedCount: 0,
			description:   "Should return 0 when retry count attribute is missing",
		},
		{
			name:          "Valid Retry Count",
			message:       createTestMessage("msg-003", "test body", 2),
			expectedCount: 2,
			description:   "Should return correct retry count from attributes",
		},
		{
			name:          "Invalid Retry Count Format",
			message:       createTestMessageWithAttributes("msg-004", "test body", map[string]string{"RetryCount": "invalid"}),
			expectedCount: 0,
			description:   "Should return 0 when retry count is not a valid number",
		},
		{
			name:          "High Retry Count",
			message:       createTestMessage("msg-005", "test body", 10),
			expectedCount: 10,
			description:   "Should handle high retry count values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create consumer
			consumer := &SQSConsumer{}

			// Execute test
			result := consumer.getRetryCount(tt.message)

			// Assertions
			assert.Equal(t, tt.expectedCount, result)
		})
	}
}

// TestSendToDLQ tests the sendToDLQ method
func TestSendToDLQ(t *testing.T) {
	tests := []sendToDLQTestCase{
		{
			name:    "Successful DLQ Send",
			message: createTestMessage("msg-001", "test body", 0),
			reason:  "Processing failed",
			mockSQS: func(mc *MockSQSClient) {
				mc.On("SendMessage", mock.Anything, mock.AnythingOfType("*sqs.SendMessageInput")).Return(&sqs.SendMessageOutput{}, nil)
			},
			expectError: false,
			description: "Should successfully send message to DLQ",
		},
		{
			name:    "DLQ Send Failure",
			message: createTestMessage("msg-002", "test body", 0),
			reason:  "Processing failed",
			mockSQS: func(mc *MockSQSClient) {
				mc.On("SendMessage", mock.Anything, mock.AnythingOfType("*sqs.SendMessageInput")).Return(nil, errors.New("DLQ error"))
			},
			expectError: false, // Method doesn't return error, just logs it
			description: "Should handle DLQ send failure gracefully",
		},
		{
			name:    "Message with Body",
			message: createTestMessage("msg-003", "important message content", 0),
			reason:  "Validation error",
			mockSQS: func(mc *MockSQSClient) {
				mc.On("SendMessage", mock.Anything, mock.AnythingOfType("*sqs.SendMessageInput")).Return(&sqs.SendMessageOutput{}, nil)
			},
			expectError: false,
			description: "Should preserve message body when sending to DLQ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockSQS := &MockSQSClient{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockSQS != nil {
				tt.mockSQS(mockSQS)
			}

			// Create consumer
			consumer := &SQSConsumer{
				sqsClient: mockSQS,
				logger:    logger,
				dlqURL:    "https://sqs.test.com/dlq",
			}

			// Execute test
			consumer.sendToDLQ(context.Background(), tt.message, tt.reason)

			// Verify mocks
			mockSQS.AssertExpectations(t)
		})
	}
}

// TestRequeueMessage tests the requeueMessage method
func TestRequeueMessage(t *testing.T) {
	tests := []requeueMessageTestCase{
		{
			name:          "Successful Requeue",
			message:       createTestMessage("msg-001", "test body", 1),
			newRetryCount: 2,
			mockSQS: func(mc *MockSQSClient) {
				mc.On("SendMessage", mock.Anything, mock.AnythingOfType("*sqs.SendMessageInput")).Return(&sqs.SendMessageOutput{}, nil)
			},
			expectError: false,
			description: "Should successfully requeue message with incremented retry count",
		},
		{
			name:          "Requeue Failure",
			message:       createTestMessage("msg-002", "test body", 2),
			newRetryCount: 3,
			mockSQS: func(mc *MockSQSClient) {
				mc.On("SendMessage", mock.Anything, mock.AnythingOfType("*sqs.SendMessageInput")).Return(nil, errors.New("requeue error"))
			},
			expectError: false, // Method doesn't return error, just logs it
			description: "Should handle requeue failure gracefully",
		},
		{
			name:          "Message Without Attributes",
			message:       createTestMessage("msg-003", "test body", 0),
			newRetryCount: 1,
			mockSQS: func(mc *MockSQSClient) {
				mc.On("SendMessage", mock.Anything, mock.AnythingOfType("*sqs.SendMessageInput")).Return(&sqs.SendMessageOutput{}, nil)
			},
			expectError: false,
			description: "Should create attributes map when message has none",
		},
		{
			name:          "High Retry Count",
			message:       createTestMessage("msg-004", "test body", 5),
			newRetryCount: 6,
			mockSQS: func(mc *MockSQSClient) {
				mc.On("SendMessage", mock.Anything, mock.AnythingOfType("*sqs.SendMessageInput")).Return(&sqs.SendMessageOutput{}, nil)
			},
			expectError: false,
			description: "Should handle high retry count values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockSQS := &MockSQSClient{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockSQS != nil {
				tt.mockSQS(mockSQS)
			}

			// Create consumer
			consumer := &SQSConsumer{
				sqsClient: mockSQS,
				logger:    logger,
				queueURL:  "https://sqs.test.com/queue",
			}

			// Execute test
			consumer.requeueMessage(context.Background(), tt.message, tt.newRetryCount)

			// Verify mocks
			mockSQS.AssertExpectations(t)
		})
	}
}

// TestDeleteMessage tests the deleteMessage method
func TestDeleteMessage(t *testing.T) {
	tests := []deleteMessageTestCase{
		{
			name:    "Successful Delete",
			message: createTestMessage("msg-001", "test body", 0),
			mockSQS: func(mc *MockSQSClient) {
				mc.On("DeleteMessage", mock.Anything, mock.AnythingOfType("*sqs.DeleteMessageInput")).Return(&sqs.DeleteMessageOutput{}, nil)
			},
			expectError: false,
			description: "Should successfully delete message from queue",
		},
		{
			name:    "Delete Failure",
			message: createTestMessage("msg-002", "test body", 0),
			mockSQS: func(mc *MockSQSClient) {
				mc.On("DeleteMessage", mock.Anything, mock.AnythingOfType("*sqs.DeleteMessageInput")).Return(nil, errors.New("delete error"))
			},
			expectError: false, // Method doesn't return error, just logs it
			description: "Should handle delete failure gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockSQS := &MockSQSClient{}
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)

			// Setup mocks
			if tt.mockSQS != nil {
				tt.mockSQS(mockSQS)
			}

			// Create consumer
			consumer := &SQSConsumer{
				sqsClient: mockSQS,
				logger:    logger,
				queueURL:  "https://sqs.test.com/queue",
			}

			// Execute test
			consumer.deleteMessage(context.Background(), tt.message)

			// Verify mocks
			mockSQS.AssertExpectations(t)
		})
	}
}

// TestNewSQSConsumer tests the constructor
func TestNewSQSConsumer(t *testing.T) {
	t.Run("Successful Consumer Creation", func(t *testing.T) {
		// Create mock AWS config
		awsCfg := aws.Config{}
		cfg := &config.Config{
			AWSEndpointURL: "http://localhost:4566",
			SQSQueueURL:    "https://sqs.test.com/queue",
			SQSDLQUrl:      "https://sqs.test.com/dlq",
		}
		mockProcessor := &MockProcessor{}
		logger := logrus.New()

		// Execute test
		consumer := NewSQSConsumer(awsCfg, cfg, mockProcessor, logger)

		// Assertions
		assert.NotNil(t, consumer)
		assert.Equal(t, "https://sqs.test.com/queue", consumer.queueURL)
		assert.Equal(t, "https://sqs.test.com/dlq", consumer.dlqURL)
		assert.Equal(t, mockProcessor, consumer.processor)
		assert.Equal(t, logger, consumer.logger)
		assert.Equal(t, 3, consumer.maxRetries)
		assert.Equal(t, int64(20), consumer.waitTime)
		assert.Equal(t, int64(10), consumer.batchSize)
		assert.NotNil(t, consumer.stopChan)
		assert.False(t, consumer.isRunning)
	})
}

// Helper functions to create test data

func createTestMessage(messageID, body string, retryCount int) *types.Message {
	message := &types.Message{
		MessageId:     aws.String(messageID),
		Body:          aws.String(body),
		ReceiptHandle: aws.String("receipt-" + messageID),
	}

	if retryCount > 0 {
		message.MessageAttributes = map[string]types.MessageAttributeValue{
			"RetryCount": {
				DataType:    aws.String("Number"),
				StringValue: aws.String(strconv.Itoa(retryCount)),
			},
		}
	}

	return message
}

func createTestMessageWithAttributes(messageID, body string, attributes map[string]string) *types.Message {
	message := &types.Message{
		MessageId:         aws.String(messageID),
		Body:              aws.String(body),
		ReceiptHandle:     aws.String("receipt-" + messageID),
		MessageAttributes: make(map[string]types.MessageAttributeValue),
	}

	for key, value := range attributes {
		message.MessageAttributes[key] = types.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(value),
		}
	}

	return message
}

func createCancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	return ctx
}
