package persistence

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sirupsen/logrus"
)

// TableNames holds the names of DynamoDB tables
type TableNames struct {
	Events        string
	EventsClients string
}

// DefaultTableNames returns default table names
func DefaultTableNames() *TableNames {
	return &TableNames{
		Events:        "events",
		EventsClients: "events-clients",
	}
}

// TableManager handles DynamoDB table creation and management
type TableManager struct {
	client     *dynamodb.Client
	tableNames *TableNames
	logger     *logrus.Logger
}

// NewTableManager creates a new table manager
func NewTableManager(awsCfg aws.Config, tableNames *TableNames, logger *logrus.Logger) *TableManager {
	return &TableManager{
		client:     dynamodb.NewFromConfig(awsCfg),
		tableNames: tableNames,
		logger:     logger,
	}
}

// CreateNewLocalTables creates local DynamoDB tables that don't already exist
func (t *TableManager) CreateNewLocalTables(ctx context.Context) error {
	// Check existing tables
	result, err := t.client.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	t.logger.WithField("existing_tables", result.TableNames).Info("Found existing tables")
	for _, foundTable := range result.TableNames {
		if foundTable == t.tableNames.Events || foundTable == t.tableNames.EventsClients {
			t.logger.WithField("table", foundTable).Info("Table already exists")
		}
	}

	// Create events table
	if err := t.createEventsTable(ctx); err != nil {
		return fmt.Errorf("failed to create events table: %w", err)
	}

	// Create events-clients table
	if err := t.createEventsClientsTable(ctx); err != nil {
		return fmt.Errorf("failed to create events-clients table: %w", err)
	}

	return nil
}

// createEventsTable creates the events table
func (t *TableManager) createEventsTable(ctx context.Context) error {
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(t.tableNames.Events),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("event_id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("client_id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("status"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("event_id"),
				KeyType:       types.KeyTypeHash,
			},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("client_id_index"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("client_id"),
						KeyType:       types.KeyTypeHash,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
			},
			{
				IndexName: aws.String("status_index"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("status"),
						KeyType:       types.KeyTypeHash,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	}

	_, err := t.client.CreateTable(ctx, input)
	if err != nil {
		return fmt.Errorf("unable to create 'events' DynamoDB table: %w", err)
	}

	t.logger.Info("Successfully created events table")
	return nil
}

// createEventsClientsTable creates the events-clients table
func (t *TableManager) createEventsClientsTable(ctx context.Context) error {
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(t.tableNames.EventsClients),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("client_id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("client_id"),
				KeyType:       types.KeyTypeHash,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	}

	_, err := t.client.CreateTable(ctx, input)
	if err != nil {
		return fmt.Errorf("unable to create 'events-clients' DynamoDB table: %w", err)
	}

	t.logger.Info("Successfully created events-clients table")
	return nil
}

// InsertSampleClientConfigs inserts sample client configurations
func (t *TableManager) InsertSampleClientConfigs(ctx context.Context) error {
	clients := []map[string]types.AttributeValue{
		{
			"client_id":     &types.AttributeValueMemberS{Value: "client-001"},
			"allowed_types": &types.AttributeValueMemberSS{Value: []string{"monitoring", "user_action", "transaction", "integration"}},
			"config": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"max_retries": &types.AttributeValueMemberS{Value: "3"},
				"timeout":     &types.AttributeValueMemberS{Value: "30s"},
			}},
			"active": &types.AttributeValueMemberBOOL{Value: true},
		},
		{
			"client_id":     &types.AttributeValueMemberS{Value: "client-002"},
			"allowed_types": &types.AttributeValueMemberSS{Value: []string{"monitoring", "user_action"}},
			"config": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"max_retries": &types.AttributeValueMemberS{Value: "5"},
				"timeout":     &types.AttributeValueMemberS{Value: "60s"},
			}},
			"active": &types.AttributeValueMemberBOOL{Value: true},
		},
		{
			"client_id":     &types.AttributeValueMemberS{Value: "client-003"},
			"allowed_types": &types.AttributeValueMemberSS{Value: []string{"transaction", "integration"}},
			"config": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"max_retries": &types.AttributeValueMemberS{Value: "2"},
				"timeout":     &types.AttributeValueMemberS{Value: "45s"},
			}},
			"active": &types.AttributeValueMemberBOOL{Value: true},
		},
	}

	for _, client := range clients {
		input := &dynamodb.PutItemInput{
			TableName: aws.String(t.tableNames.EventsClients),
			Item:      client,
		}

		_, err := t.client.PutItem(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to insert client config %s: %w", client["client_id"], err)
		}
	}

	t.logger.Info("Successfully inserted sample client configurations")
	return nil
}
