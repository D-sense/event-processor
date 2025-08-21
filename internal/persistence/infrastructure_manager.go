package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/sirupsen/logrus"
)

// InfrastructureManager coordinates the creation of all required infrastructure
type InfrastructureManager struct {
	tableManager *TableManager
	queueManager *QueueManager
	logger       *logrus.Logger
}

// NewInfrastructureManager creates a new infrastructure manager
func NewInfrastructureManager(awsCfg aws.Config, logger *logrus.Logger) *InfrastructureManager {
	tableNames := DefaultTableNames()
	queueNames := DefaultQueueNames()

	return &InfrastructureManager{
		tableManager: NewTableManager(awsCfg, tableNames, logger),
		queueManager: NewQueueManager(awsCfg, queueNames, logger),
		logger:       logger,
	}
}

// SetupInfrastructure creates all required infrastructure (tables and queues)
func (i *InfrastructureManager) SetupInfrastructure(ctx context.Context) error {
	i.logger.Info("Starting infrastructure setup...")

	// Create DynamoDB tables
	i.logger.Info("Setting up DynamoDB tables...")
	if err := i.tableManager.CreateNewLocalTables(ctx); err != nil {
		return fmt.Errorf("failed to setup DynamoDB tables: %w", err)
	}

	// Wait a bit for tables to be active
	i.logger.Info("Waiting for DynamoDB tables to be active...")
	time.Sleep(5 * time.Second)

	// Insert sample client configurations
	i.logger.Info("Inserting sample client configurations...")
	if err := i.tableManager.InsertSampleClientConfigs(ctx); err != nil {
		i.logger.WithError(err).Warn("Failed to insert sample client configs, continuing...")
	}

	// Create SQS queues
	i.logger.Info("Setting up SQS queues...")
	if err := i.queueManager.CreateNewLocalQueues(ctx); err != nil {
		return fmt.Errorf("failed to setup SQS queues: %w", err)
	}

	// Wait a bit for queues to be ready
	i.logger.Info("Waiting for SQS queues to be ready...")
	time.Sleep(3 * time.Second)

	// Get queue URLs for verification
	queueURLs, err := i.queueManager.GetQueueURLs(ctx)
	if err != nil {
		i.logger.WithError(err).Warn("Failed to get queue URLs, continuing...")
	} else {
		i.logger.WithField("queue_urls", queueURLs).Info("Successfully retrieved queue URLs")
	}

	i.logger.Info("Infrastructure setup completed successfully!")
	return nil
}
