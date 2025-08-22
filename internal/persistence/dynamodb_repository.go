package persistence

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/d-sense/event-processor/internal/config"
	"github.com/d-sense/event-processor/pkg/models"
)

// DynamoDBClient defines the interface for DynamoDB operations
type DynamoDBClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error)
	ListTables(ctx context.Context, params *dynamodb.ListTablesInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error)
	CreateTable(ctx context.Context, params *dynamodb.CreateTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.CreateTableOutput, error)
}

// DynamoDBRepository implements Repository interface using AWS DynamoDB
type DynamoDBRepository struct {
	client    DynamoDBClient
	tableName string
}

// NewDynamoDBRepository creates a new DynamoDB repository
func NewDynamoDBRepository(awsCfg aws.Config, cfg *config.Config) Repository {
	return &DynamoDBRepository{
		client:    dynamodb.NewFromConfig(awsCfg),
		tableName: cfg.DynamoDBTableName,
	}
}

// marshalPayload converts a map[string]interface{} to DynamoDB attribute values
func (r *DynamoDBRepository) marshalPayload(payload map[string]interface{}) map[string]types.AttributeValue {
	result := make(map[string]types.AttributeValue)

	for key, value := range payload {
		switch v := value.(type) {
		case string:
			result[key] = &types.AttributeValueMemberS{Value: v}
		case int:
			result[key] = &types.AttributeValueMemberN{Value: strconv.Itoa(v)}
		case int64:
			result[key] = &types.AttributeValueMemberN{Value: strconv.FormatInt(v, 10)}
		case float64:
			result[key] = &types.AttributeValueMemberN{Value: strconv.FormatFloat(v, 'f', -1, 64)}
		case bool:
			result[key] = &types.AttributeValueMemberBOOL{Value: v}
		case map[string]interface{}:
			result[key] = &types.AttributeValueMemberM{Value: r.marshalPayload(v)}
		case []string:
			result[key] = &types.AttributeValueMemberSS{Value: v}
		default:
			// Convert unknown types to string
			result[key] = &types.AttributeValueMemberS{Value: fmt.Sprintf("%v", v)}
		}
	}

	return result
}

// SaveEvent saves an event to DynamoDB
func (r *DynamoDBRepository) SaveEvent(ctx context.Context, event *models.ProcessedEvent) error {
	// Manually create the item with correct DynamoDB attribute names
	item := map[string]types.AttributeValue{
		"event_id":     &types.AttributeValueMemberS{Value: event.EventID},
		"event_type":   &types.AttributeValueMemberS{Value: string(event.EventType)},
		"client_id":    &types.AttributeValueMemberS{Value: event.ClientID},
		"timestamp":    &types.AttributeValueMemberS{Value: event.Timestamp.Format(time.RFC3339)},
		"payload":      &types.AttributeValueMemberM{Value: r.marshalPayload(event.Payload)},
		"version":      &types.AttributeValueMemberS{Value: event.Version},
		"processed_at": &types.AttributeValueMemberS{Value: event.ProcessedAt.Format(time.RFC3339)},
		"status":       &types.AttributeValueMemberS{Value: string(event.Status)},
		"retry_count":  &types.AttributeValueMemberN{Value: strconv.Itoa(event.RetryCount)},
		"ttl":          &types.AttributeValueMemberN{Value: strconv.FormatInt(event.TTL, 10)},
	}

	// Add error message if present
	if event.ErrorMsg != "" {
		item["error_msg"] = &types.AttributeValueMemberS{Value: event.ErrorMsg}
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	_, err := r.client.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to save event to DynamoDB: %w", err)
	}

	return nil
}

// HealthCheck performs a health check on the DynamoDB connection
func (r *DynamoDBRepository) HealthCheck(ctx context.Context) error {
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String(r.tableName),
	}

	_, err := r.client.DescribeTable(ctx, input)
	return err
}

// GetClientConfig retrieves client configuration
func (r *DynamoDBRepository) GetClientConfig(ctx context.Context, clientID string) (*models.ClientConfig, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String("events-clients"),
		Key: map[string]types.AttributeValue{
			"client_id": &types.AttributeValueMemberS{Value: clientID},
		},
	}

	result, err := r.client.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get client config: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("client config not found: %s", clientID)
	}

	// Manually extract attributes to avoid unmarshaling issues
	config := &models.ClientConfig{
		ClientID: clientID,
		Active:   true, // Default to true
	}

	// Extract allowed types
	if allowedTypesAttr, ok := result.Item["allowed_types"]; ok {
		if allowedTypesSS, ok := allowedTypesAttr.(*types.AttributeValueMemberSS); ok {
			config.AllowedTypes = make([]models.EventType, len(allowedTypesSS.Value))
			for i, eventTypeStr := range allowedTypesSS.Value {
				config.AllowedTypes[i] = models.EventType(eventTypeStr)
			}
		}
	}

	// Extract active status
	if activeAttr, ok := result.Item["active"]; ok {
		if activeBool, ok := activeAttr.(*types.AttributeValueMemberBOOL); ok {
			config.Active = activeBool.Value
		}
	}

	// Extract config map
	if configAttr, ok := result.Item["config"]; ok {
		if configMap, ok := configAttr.(*types.AttributeValueMemberM); ok {
			config.Config = make(map[string]string)
			for key, value := range configMap.Value {
				if strValue, ok := value.(*types.AttributeValueMemberS); ok {
					config.Config[key] = strValue.Value
				}
			}
		}
	}

	return config, nil
}
