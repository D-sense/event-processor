package persistence

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/sirupsen/logrus"
)

// QueueNames holds the names of SQS queues
type QueueNames struct {
	EventQueue string
	EventDLQ   string
}

// DefaultQueueNames returns default queue names
func DefaultQueueNames() *QueueNames {
	return &QueueNames{
		EventQueue: "event-queue",
		EventDLQ:   "event-dlq",
	}
}

// QueueManager handles SQS queue creation and management
type QueueManager struct {
	client     *sqs.Client
	queueNames *QueueNames
	logger     *logrus.Logger
}

// NewQueueManager creates a new queue manager
func NewQueueManager(awsCfg aws.Config, queueNames *QueueNames, logger *logrus.Logger) *QueueManager {
	return &QueueManager{
		client:     sqs.NewFromConfig(awsCfg),
		queueNames: queueNames,
		logger:     logger,
	}
}

// CreateNewLocalQueues creates local SQS queues that don't already exist
func (q *QueueManager) CreateNewLocalQueues(ctx context.Context) error {
	// Check existing queues
	result, err := q.client.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		return fmt.Errorf("failed to list queues: %w", err)
	}

	q.logger.WithField("existing_queues", result.QueueUrls).Info("Found existing queues")

	// Create event queue
	if err := q.createEventQueue(ctx); err != nil {
		return fmt.Errorf("failed to create event queue: %w", err)
	}

	// Create event DLQ
	if err := q.createEventDLQ(ctx); err != nil {
		return fmt.Errorf("failed to create event DLQ: %w", err)
	}

	return nil
}

// createEventQueue creates the main event queue
func (q *QueueManager) createEventQueue(ctx context.Context) error {
	attributes := map[string]string{
		"MessageRetentionPeriod": "1209600", // 14 days
		"VisibilityTimeout":      "30",      // 30 seconds
	}

	input := &sqs.CreateQueueInput{
		QueueName:  aws.String(q.queueNames.EventQueue),
		Attributes: attributes,
	}

	_, err := q.client.CreateQueue(ctx, input)
	if err != nil {
		return fmt.Errorf("unable to create event queue: %w", err)
	}

	q.logger.Info("Successfully created event queue")
	return nil
}

// createEventDLQ creates the dead letter queue for failed events
func (q *QueueManager) createEventDLQ(ctx context.Context) error {
	attributes := map[string]string{
		"MessageRetentionPeriod": "1209600", // 14 days
		"VisibilityTimeout":      "30",      // 30 seconds
	}

	input := &sqs.CreateQueueInput{
		QueueName:  aws.String(q.queueNames.EventDLQ),
		Attributes: attributes,
	}

	_, err := q.client.CreateQueue(ctx, input)
	if err != nil {
		return fmt.Errorf("unable to create event DLQ: %w", err)
	}

	q.logger.Info("Successfully created event DLQ")
	return nil
}

// GetQueueURLs returns the URLs for the created queues
func (q *QueueManager) GetQueueURLs(ctx context.Context) (map[string]string, error) {
	urls := make(map[string]string)

	// Get event queue URL
	result, err := q.client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(q.queueNames.EventQueue),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get event queue URL: %w", err)
	}
	urls["event_queue"] = *result.QueueUrl

	// Get event DLQ URL
	result, err = q.client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(q.queueNames.EventDLQ),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get event DLQ URL: %w", err)
	}
	urls["event_dlq"] = *result.QueueUrl

	return urls, nil
}
