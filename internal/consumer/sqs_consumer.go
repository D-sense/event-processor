package consumer

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/sirupsen/logrus"

	"github.com/d-sense/event-processor/internal/config"
	"github.com/d-sense/event-processor/internal/processor"
	"github.com/d-sense/event-processor/pkg/logger"
)

// SQSClient defines the interface for SQS operations
type SQSClient interface {
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

// SQSConsumer handles consuming messages from AWS SQS
type SQSConsumer struct {
	sqsClient  SQSClient
	queueURL   string
	dlqURL     string
	processor  processor.Processor
	logger     *logrus.Logger
	stopChan   chan struct{}
	isRunning  bool
	maxRetries int
	waitTime   int64
	batchSize  int64
}

// NewSQSConsumer creates a new SQS consumer
func NewSQSConsumer(awsCfg aws.Config, cfg *config.Config, processor processor.Processor, logger *logrus.Logger) *SQSConsumer {
	// Create SQS client with proper LocalStack 3.x configuration
	sqsClient := sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(cfg.AWSEndpointURL)
	})

	return &SQSConsumer{
		sqsClient:  sqsClient,
		queueURL:   cfg.SQSQueueURL,
		dlqURL:     cfg.SQSDLQUrl,
		processor:  processor,
		logger:     logger,
		stopChan:   make(chan struct{}),
		maxRetries: 3,  // Default max retries
		waitTime:   20, // Long polling
		batchSize:  10,
	}
}

// Start begins consuming messages from SQS
func (c *SQSConsumer) Start(ctx context.Context) error {
	if c.isRunning {
		return fmt.Errorf("consumer is already running")
	}

	c.isRunning = true
	c.logger.Info("Starting SQS consumer")

	go c.consumeMessages(ctx)

	return nil
}

// Stop gracefully stops the consumer
func (c *SQSConsumer) Stop(ctx context.Context) error {
	if !c.isRunning {
		return nil
	}

	c.logger.Info("Stopping SQS consumer")
	c.isRunning = false
	close(c.stopChan)

	// Wait for context cancellation or timeout
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for consumer to stop")
	}
}

// consumeMessages continuously polls SQS for messages
func (c *SQSConsumer) consumeMessages(ctx context.Context) {
	for {
		select {
		case <-c.stopChan:
			c.logger.Info("SQS consumer stopped")
			return
		default:
			c.pollMessages(ctx)
		}
	}
}

// pollMessages retrieves and processes a batch of messages
func (c *SQSConsumer) pollMessages(ctx context.Context) {
	input := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(c.queueURL),
		MaxNumberOfMessages: int32(c.batchSize),
		WaitTimeSeconds:     int32(c.waitTime),
		MessageAttributeNames: []string{
			"All",
		},
	}

	result, err := c.sqsClient.ReceiveMessage(ctx, input)
	if err != nil {
		c.logger.WithError(err).Error("Failed to receive messages from SQS")
		return
	}

	if len(result.Messages) == 0 {
		return
	}

	c.logger.WithField("message_count", len(result.Messages)).Debug("Received messages from SQS")

	for _, message := range result.Messages {
		go c.processMessage(ctx, &message)
	}
}

// processMessage processes a single SQS message
func (c *SQSConsumer) processMessage(ctx context.Context, message *types.Message) {
	messageID := aws.ToString(message.MessageId)
	receiptHandle := aws.ToString(message.ReceiptHandle)

	// Create logger with message context
	logger := logger.WithFields(c.logger, map[string]interface{}{
		"message_id":     messageID,
		"receipt_handle": receiptHandle,
	})
	logger.Debug("Processing message")

	// Check retry count
	retryCount := c.getRetryCount(message)
	if retryCount >= c.maxRetries {
		logger.WithField("retry_count", retryCount).Warn("Message exceeded max retries, sending to DLQ")
		c.sendToDLQ(ctx, message, "Max retries exceeded")
		c.deleteMessage(ctx, message)
		return
	}

	// Process the message
	if err := c.processor.ProcessEvent(context.Background(), message); err != nil {
		logger.WithError(err).Error("Failed to process event")

		// Increment retry count and requeue if under max retries
		if retryCount < c.maxRetries {
			c.requeueMessage(ctx, message, retryCount+1)
		} else {
			c.sendToDLQ(ctx, message, fmt.Sprintf("Processing failed: %v", err))
		}
		return
	}

	// Successfully processed, delete the message
	logger.Info("Successfully processed message")
	c.deleteMessage(ctx, message)
}

// sendToDLQ sends a message to the Dead Letter Queue
func (c *SQSConsumer) sendToDLQ(ctx context.Context, message *types.Message, reason string) {
	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(c.dlqURL),
		MessageBody: message.Body,
		MessageAttributes: map[string]types.MessageAttributeValue{
			"OriginalMessageId": {
				DataType:    aws.String("String"),
				StringValue: message.MessageId,
			},
			"FailureReason": {
				DataType:    aws.String("String"),
				StringValue: aws.String(reason),
			},
		},
	}

	_, err := c.sqsClient.SendMessage(ctx, input)
	if err != nil {
		c.logger.WithError(err).Error("Failed to send message to DLQ")
	}
}

// requeueMessage puts a message back in the queue with incremented retry count
func (c *SQSConsumer) requeueMessage(ctx context.Context, message *types.Message, newRetryCount int) {
	// Update retry count in message attributes
	if message.MessageAttributes == nil {
		message.MessageAttributes = make(map[string]types.MessageAttributeValue)
	}

	message.MessageAttributes["RetryCount"] = types.MessageAttributeValue{
		DataType:    aws.String("Number"),
		StringValue: aws.String(strconv.Itoa(newRetryCount)),
	}

	// Send back to queue
	input := &sqs.SendMessageInput{
		QueueUrl:          aws.String(c.queueURL),
		MessageBody:       message.Body,
		MessageAttributes: message.MessageAttributes,
		DelaySeconds:      int32(newRetryCount * 5), // Exponential backoff
	}

	_, err := c.sqsClient.SendMessage(ctx, input)
	if err != nil {
		c.logger.WithError(err).Error("Failed to requeue message")
	} else {
		// Create logger with requeue context
		requeueLogger := logger.WithFields(c.logger, map[string]interface{}{
			"message_id":  aws.ToString(message.MessageId),
			"retry_count": newRetryCount,
		})
		requeueLogger.Info("Message requeued")
	}
}

// deleteMessage removes a processed message from the queue
func (c *SQSConsumer) deleteMessage(ctx context.Context, message *types.Message) {
	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.queueURL),
		ReceiptHandle: message.ReceiptHandle,
	}

	_, err := c.sqsClient.DeleteMessage(ctx, input)
	if err != nil {
		c.logger.WithError(err).Error("Failed to delete message from queue")
	}
}

// getRetryCount extracts the retry count from message attributes
func (c *SQSConsumer) getRetryCount(message *types.Message) int {
	if message.MessageAttributes == nil {
		return 0
	}

	retryAttr, exists := message.MessageAttributes["RetryCount"]
	if !exists {
		return 0
	}

	if retryAttr.StringValue != nil {
		if count, err := strconv.Atoi(*retryAttr.StringValue); err == nil {
			return count
		}
	}

	return 0
}
