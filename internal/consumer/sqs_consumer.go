package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sirupsen/logrus"

	"github.com/d-sense/event-processor/internal/config"
	"github.com/d-sense/event-processor/internal/processor"
)

const (
	eventProcessorMaxRetries = 3 // Maximum retries for processing an event
)

// SQSConsumer handles consuming messages from AWS SQS
type SQSConsumer struct {
	sqsClient   *sqs.SQS
	queueURL    string
	dlqURL      string
	processor   *processor.EventProcessor
	config      *config.Config
	logger      *logrus.Logger
	workerPool  chan struct{}
	maxMessages int64
	waitTime    int64
}

// NewSQSConsumer creates a new SQS consumer
func NewSQSConsumer(sess *session.Session, cfg *config.Config, proc *processor.EventProcessor, logger *logrus.Logger) *SQSConsumer {
	sqsClient := sqs.New(sess, &aws.Config{
		Endpoint: aws.String(cfg.DynamoDBEndpoint), // Use same endpoint for SQS in LocalStack
	})

	return &SQSConsumer{
		sqsClient:   sqsClient,
		queueURL:    cfg.SQSQueueURL,
		dlqURL:      cfg.SQSDLQUrl,
		processor:   proc,
		config:      cfg,
		logger:      logger,
		workerPool:  make(chan struct{}, cfg.WorkerPoolSize),
		maxMessages: cfg.SQSMaxMessages,
		waitTime:    cfg.SQSWaitTimeSeconds,
	}
}

// Start begins consuming messages from the queue
func (c *SQSConsumer) Start(ctx context.Context) error {
	c.logger.Info("Starting SQS consumer")

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("SQS consumer context cancelled")
			return nil
		default:
			if err := c.pollMessages(ctx); err != nil {
				c.logger.WithError(err).Error("Error polling messages")
				time.Sleep(5 * time.Second) // Wait before retrying
			}
		}
	}
}

// pollMessages polls for messages from SQS
func (c *SQSConsumer) pollMessages(ctx context.Context) error {
	input := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(c.queueURL),
		MaxNumberOfMessages: aws.Int64(c.maxMessages),
		WaitTimeSeconds:     aws.Int64(c.waitTime),
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
	}

	result, err := c.sqsClient.ReceiveMessageWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to receive messages: %w", err)
	}

	if len(result.Messages) == 0 {
		c.logger.Debug("No messages received")
		return nil
	}

	c.logger.WithField("count", len(result.Messages)).Debug("Received messages")

	// We process the messages concurrently
	var wg sync.WaitGroup
	for _, message := range result.Messages {
		select {
		case c.workerPool <- struct{}{}: // Acquire worker
			wg.Add(1)
			go func(msg *sqs.Message) {
				defer func() {
					<-c.workerPool // Release worker
					wg.Done()
				}()
				c.processMessage(ctx, msg)
			}(message)
		case <-ctx.Done():
			c.logger.Info("Context cancelled while processing messages")
			return nil
		}
	}

	wg.Wait()
	return nil
}

// processMessage processes a single SQS message
func (c *SQSConsumer) processMessage(ctx context.Context, message *sqs.Message) {
	messageID := aws.StringValue(message.MessageId)
	receiptHandle := aws.StringValue(message.ReceiptHandle)

	logger := c.logger.WithFields(logrus.Fields{
		"message_id":     messageID,
		"receipt_handle": receiptHandle,
	})

	logger.Debug("Processing message")

	// Parse message body
	var eventData interface{}
	if err := json.Unmarshal([]byte(aws.StringValue(message.Body)), &eventData); err != nil {
		logger.WithError(err).Error("Failed to unmarshal message body")
		c.sendToDLQ(ctx, message, fmt.Sprintf("JSON unmarshal error: %v", err))
		c.deleteMessage(ctx, receiptHandle, logger)
		return
	}

	// Process the event
	if err := c.processor.ProcessEvent(ctx, eventData); err != nil {
		logger.WithError(err).Error("Failed to process event")

		// Check retry count from message attributes
		retryCount := c.getRetryCount(message)
		if retryCount >= eventProcessorMaxRetries { // Max retries
			c.sendToDLQ(ctx, message, fmt.Sprintf("Max retries exceeded: %v", err))
			c.deleteMessage(ctx, receiptHandle, logger)
		} else {
			// Re-queue with increased retry count
			c.requeueMessage(ctx, message, retryCount+1, err.Error())
			c.deleteMessage(ctx, receiptHandle, logger)
		}
		return
	}

	// Successfully processed, delete from queue
	c.deleteMessage(ctx, receiptHandle, logger)
	logger.Info("Successfully processed and deleted message")
}

// deleteMessage deletes a message from the queue
func (c *SQSConsumer) deleteMessage(ctx context.Context, receiptHandle string, logger *logrus.Entry) {
	_, err := c.sqsClient.DeleteMessageWithContext(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	if err != nil {
		logger.WithError(err).Error("Failed to delete message")
	}
}

// sendToDLQ sends a message to the dead letter queue
func (c *SQSConsumer) sendToDLQ(ctx context.Context, originalMessage *sqs.Message, reason string) {
	dlqMessage := map[string]interface{}{
		"originalBody": aws.StringValue(originalMessage.Body),
		"errorReason":  reason,
		"timestamp":    time.Now().UTC(),
		"messageId":    aws.StringValue(originalMessage.MessageId),
	}

	dlqBody, _ := json.Marshal(dlqMessage)

	_, err := c.sqsClient.SendMessageWithContext(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(c.dlqURL),
		MessageBody: aws.String(string(dlqBody)),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"ErrorReason": {
				DataType:    aws.String("String"),
				StringValue: aws.String(reason),
			},
		},
	})

	if err != nil {
		c.logger.WithError(err).Error("Failed to send message to DLQ")
	} else {
		c.logger.WithField("reason", reason).Info("Sent message to DLQ")
	}
}

// requeueMessage re-queues a message with retry count
func (c *SQSConsumer) requeueMessage(ctx context.Context, message *sqs.Message, retryCount int, errorMsg string) {
	delay := c.calculateDelay(retryCount)

	_, err := c.sqsClient.SendMessageWithContext(ctx, &sqs.SendMessageInput{
		QueueUrl:     aws.String(c.queueURL),
		MessageBody:  message.Body,
		DelaySeconds: aws.Int64(int64(delay.Seconds())),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"RetryCount": {
				DataType:    aws.String("Number"),
				StringValue: aws.String(fmt.Sprintf("%d", retryCount)),
			},
			"LastError": {
				DataType:    aws.String("String"),
				StringValue: aws.String(errorMsg),
			},
		},
	})

	if err != nil {
		c.logger.WithError(err).Error("Failed to requeue message")
	} else {
		c.logger.WithFields(logrus.Fields{
			"retry_count": retryCount,
			"delay":       delay,
		}).Info("Requeued message with delay")
	}
}

// getRetryCount extracts retry count from message attributes
func (c *SQSConsumer) getRetryCount(message *sqs.Message) int {
	if attrs := message.MessageAttributes; attrs != nil {
		if retryAttr := attrs["RetryCount"]; retryAttr != nil {
			if retryCount := aws.StringValue(retryAttr.StringValue); retryCount != "" {
				var count int
				fmt.Sscanf(retryCount, "%d", &count)
				return count
			}
		}
	}
	return 0
}

// calculateDelay calculates exponential backoff delay
func (c *SQSConsumer) calculateDelay(retryCount int) time.Duration {
	// Exponential backoff: 2^retryCount seconds, max 300 seconds (5 minutes)
	delay := time.Duration(1<<retryCount) * time.Second
	if delay > 300*time.Second {
		delay = 300 * time.Second
	}
	return delay
}
