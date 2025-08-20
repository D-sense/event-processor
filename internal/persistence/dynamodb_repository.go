package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"github.com/d-sense/event-processor/internal/config"
	"github.com/d-sense/event-processor/pkg/models"
)

// DynamoDBRepository implements Repository interface using DynamoDB
type DynamoDBRepository struct {
	client    *dynamodb.DynamoDB
	tableName string
}

// NewDynamoDBRepository creates a new DynamoDB repository
func NewDynamoDBRepository(sess *session.Session, cfg *config.Config) Repository {
	dynamoConfig := &aws.Config{}
	if cfg.DynamoDBEndpoint != "" {
		dynamoConfig.Endpoint = aws.String(cfg.DynamoDBEndpoint)
	}

	client := dynamodb.New(sess, dynamoConfig)

	return &DynamoDBRepository{
		client:    client,
		tableName: cfg.DynamoDBTableName,
	}
}

// SaveEvent saves a processed event to DynamoDB
func (r *DynamoDBRepository) SaveEvent(ctx context.Context, event *models.ProcessedEvent) error {
	// Marshal the event to DynamoDB attribute values
	item, err := dynamodbattribute.MarshalMap(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Put item in DynamoDB
	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	_, err = r.client.PutItemWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to save event to DynamoDB: %w", err)
	}

	return nil
}

// GetEvent retrieves an event by ID from DynamoDB
func (r *DynamoDBRepository) GetEvent(ctx context.Context, eventID string) (*models.ProcessedEvent, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"event_id": {
				S: aws.String(eventID),
			},
		},
	}

	result, err := r.client.GetItemWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get event from DynamoDB: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("event not found: %s", eventID)
	}

	var event models.ProcessedEvent
	if err := dynamodbattribute.UnmarshalMap(result.Item, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	return &event, nil
}

// GetEventsByClient retrieves events for a specific client
func (r *DynamoDBRepository) GetEventsByClient(ctx context.Context, clientID string, limit int) ([]*models.ProcessedEvent, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("client-id-index"), // Assumes GSI exists
		KeyConditionExpression: aws.String("client_id = :client_id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":client_id": {
				S: aws.String(clientID),
			},
		},
		Limit:            aws.Int64(int64(limit)),
		ScanIndexForward: aws.Bool(false), // Most recent first
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by client: %w", err)
	}

	var events []*models.ProcessedEvent
	for _, item := range result.Items {
		var event models.ProcessedEvent
		if err := dynamodbattribute.UnmarshalMap(item, &event); err != nil {
			continue // Skip malformed items
		}
		events = append(events, &event)
	}

	return events, nil
}

// GetEventsByStatus retrieves events by status
func (r *DynamoDBRepository) GetEventsByStatus(ctx context.Context, status models.EventStatus, limit int) ([]*models.ProcessedEvent, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("status-index"), // Assumes GSI exists
		KeyConditionExpression: aws.String("#status = :status"),
		ExpressionAttributeNames: map[string]*string{
			"#status": aws.String("status"), // 'status' is a reserved word
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":status": {
				S: aws.String(string(status)),
			},
		},
		Limit:            aws.Int64(int64(limit)),
		ScanIndexForward: aws.Bool(false),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by status: %w", err)
	}

	var events []*models.ProcessedEvent
	for _, item := range result.Items {
		var event models.ProcessedEvent
		if err := dynamodbattribute.UnmarshalMap(item, &event); err != nil {
			continue
		}
		events = append(events, &event)
	}

	return events, nil
}

// UpdateEventStatus updates the status of an event
func (r *DynamoDBRepository) UpdateEventStatus(ctx context.Context, eventID string, status models.EventStatus, errorMsg string) error {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"event_id": {
				S: aws.String(eventID),
			},
		},
		UpdateExpression: aws.String("SET #status = :status, processed_at = :processed_at"),
		ExpressionAttributeNames: map[string]*string{
			"#status": aws.String("status"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":status": {
				S: aws.String(string(status)),
			},
			":processed_at": {
				S: aws.String(time.Now().UTC().Format(time.RFC3339)),
			},
		},
	}

	// Add error message if provided
	if errorMsg != "" {
		input.UpdateExpression = aws.String("SET #status = :status, processed_at = :processed_at, error_msg = :error_msg")
		input.ExpressionAttributeValues[":error_msg"] = &dynamodb.AttributeValue{
			S: aws.String(errorMsg),
		}
	}

	_, err := r.client.UpdateItemWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update event status: %w", err)
	}

	return nil
}

// SaveClientConfig saves client configuration
func (r *DynamoDBRepository) SaveClientConfig(ctx context.Context, config *models.ClientConfig) error {
	item, err := dynamodbattribute.MarshalMap(config)
	if err != nil {
		return fmt.Errorf("failed to marshal client config: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName + "-clients"), // Separate table for client configs
		Item:      item,
	}

	_, err = r.client.PutItemWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to save client config: %w", err)
	}

	return nil
}

// GetClientConfig retrieves client configuration
func (r *DynamoDBRepository) GetClientConfig(ctx context.Context, clientID string) (*models.ClientConfig, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName + "-clients"),
		Key: map[string]*dynamodb.AttributeValue{
			"client_id": {
				S: aws.String(clientID),
			},
		},
	}

	result, err := r.client.GetItemWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get client config: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("client config not found: %s", clientID)
	}

	var config models.ClientConfig
	if err := dynamodbattribute.UnmarshalMap(result.Item, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal client config: %w", err)
	}

	return &config, nil
}

// HealthCheck performs a health check on the DynamoDB connection
func (r *DynamoDBRepository) HealthCheck(ctx context.Context) error {
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String(r.tableName),
	}

	_, err := r.client.DescribeTableWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("DynamoDB health check failed: %w", err)
	}

	return nil
}

// DeleteExpiredEvents deletes events that have exceeded their TTL
func (r *DynamoDBRepository) DeleteExpiredEvents(ctx context.Context) error {
	// DynamoDB handles TTL automatically, but we can scan for expired items if needed
	currentTime := time.Now().Unix()

	input := &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("ttl < :current_time"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":current_time": {
				N: aws.String(fmt.Sprintf("%d", currentTime)),
			},
		},
	}

	result, err := r.client.ScanWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to scan expired events: %w", err)
	}

	// Batch delete expired items
	if len(result.Items) > 0 {
		return r.batchDeleteItems(ctx, result.Items)
	}

	return nil
}

// batchDeleteItems performs batch delete operation
func (r *DynamoDBRepository) batchDeleteItems(ctx context.Context, items []map[string]*dynamodb.AttributeValue) error {
	const maxBatchSize = 25 // DynamoDB batch write limit

	for i := 0; i < len(items); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(items) {
			end = len(items)
		}

		var writeRequests []*dynamodb.WriteRequest
		for _, item := range items[i:end] {
			writeRequests = append(writeRequests, &dynamodb.WriteRequest{
				DeleteRequest: &dynamodb.DeleteRequest{
					Key: map[string]*dynamodb.AttributeValue{
						"event_id": item["event_id"],
					},
				},
			})
		}

		input := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				r.tableName: writeRequests,
			},
		}

		_, err := r.client.BatchWriteItemWithContext(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to batch delete items: %w", err)
		}
	}

	return nil
}

// GetEventStats returns statistics about events
func (r *DynamoDBRepository) GetEventStats(ctx context.Context, clientID string, hours int) (*models.EventMetrics, error) {
	// Calculate time range
	endTime := time.Now().UTC()
	startTime := endTime.Add(-time.Duration(hours) * time.Hour)

	// Query events in time range
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("client-id-timestamp-index"), // Assumes composite GSI exists
		KeyConditionExpression: aws.String("client_id = :client_id AND #timestamp BETWEEN :start_time AND :end_time"),
		ExpressionAttributeNames: map[string]*string{
			"#timestamp": aws.String("timestamp"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":client_id": {
				S: aws.String(clientID),
			},
			":start_time": {
				S: aws.String(startTime.Format(time.RFC3339)),
			},
			":end_time": {
				S: aws.String(endTime.Format(time.RFC3339)),
			},
		},
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get event stats: %w", err)
	}

	// Calculate metrics
	var totalProcessed, totalFailed int64
	var totalLatency time.Duration

	for _, item := range result.Items {
		var event models.ProcessedEvent
		if err := dynamodbattribute.UnmarshalMap(item, &event); err != nil {
			continue
		}

		if event.Status == models.EventStatusProcessed {
			totalProcessed++
		} else if event.Status == models.EventStatusFailed {
			totalFailed++
		}

		// Calculate processing latency
		if !event.ProcessedAt.IsZero() && !event.Timestamp.IsZero() {
			latency := event.ProcessedAt.Sub(event.Timestamp)
			totalLatency += latency
		}
	}

	totalEvents := int64(len(result.Items))
	var avgLatencyMs int64
	if totalEvents > 0 {
		avgLatencyMs = totalLatency.Milliseconds() / totalEvents
	}

	processingRate := totalProcessed
	if hours > 0 {
		processingRate = totalProcessed / int64(hours)
	}

	return &models.EventMetrics{
		TotalProcessed:   totalProcessed,
		TotalFailed:      totalFailed,
		ProcessingRate:   processingRate,
		AverageLatencyMs: avgLatencyMs,
	}, nil
}
