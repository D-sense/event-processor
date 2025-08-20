#!/bin/bash

set -e

echo "Initializing LocalStack resources..."

# Wait for LocalStack to be ready
echo "Waiting for LocalStack to be ready..."
while ! curl -s http://localhost:4566/_localstack/health | grep -q "running"; do
    echo "Waiting for LocalStack..."
    sleep 2
done

echo "LocalStack is ready. Creating resources..."

# Create SQS Queues
echo "Creating SQS queues..."

# Create main event queue
awslocal sqs create-queue \
    --queue-name event-queue \
    --attributes VisibilityTimeoutSeconds=300,MessageRetentionPeriod=1209600

# Create dead letter queue
awslocal sqs create-queue \
    --queue-name event-dlq \
    --attributes MessageRetentionPeriod=1209600

echo "SQS queues created successfully"

# Create DynamoDB Tables
echo "Creating DynamoDB tables..."

# Create main events table
awslocal dynamodb create-table \
    --table-name events \
    --attribute-definitions \
        AttributeName=event_id,AttributeType=S \
        AttributeName=client_id,AttributeType=S \
        AttributeName=timestamp,AttributeType=S \
        AttributeName=status,AttributeType=S \
    --key-schema \
        AttributeName=event_id,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST \
    --global-secondary-indexes \
        'IndexName=client-id-index,KeySchema=[{AttributeName=client_id,KeyType=HASH}],Projection={ProjectionType=ALL}' \
        'IndexName=status-index,KeySchema=[{AttributeName=status,KeyType=HASH}],Projection={ProjectionType=ALL}' \
        'IndexName=client-id-timestamp-index,KeySchema=[{AttributeName=client_id,KeyType=HASH},{AttributeName=timestamp,KeyType=RANGE}],Projection={ProjectionType=ALL}'

# Enable TTL on the events table
awslocal dynamodb update-time-to-live \
    --table-name events \
    --time-to-live-specification Enabled=true,AttributeName=ttl

# Create client configurations table
awslocal dynamodb create-table \
    --table-name events-clients \
    --attribute-definitions \
        AttributeName=client_id,AttributeType=S \
    --key-schema \
        AttributeName=client_id,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST

echo "DynamoDB tables created successfully"

# Insert sample client configurations
echo "Inserting sample client configurations..."

awslocal dynamodb put-item \
    --table-name events-clients \
    --item '{
        "client_id": {"S": "client-123"},
        "allowed_types": {"SS": ["monitoring", "user_action", "transaction"]},
        "config": {"M": {"max_events_per_hour": {"S": "1000"}}},
        "active": {"BOOL": true}
    }'

awslocal dynamodb put-item \
    --table-name events-clients \
    --item '{
        "client_id": {"S": "client-456"},
        "allowed_types": {"SS": ["integration", "monitoring"]},
        "config": {"M": {"max_events_per_hour": {"S": "500"}}},
        "active": {"BOOL": true}
    }'

echo "Sample client configurations inserted"

# Verify resources
echo "Verifying created resources..."

echo "SQS Queues:"
awslocal sqs list-queues

echo "DynamoDB Tables:"
awslocal dynamodb list-tables

echo "LocalStack initialization completed successfully!"