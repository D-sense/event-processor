package config

import (
	"os"
	"strconv"
)

type Config struct {
	// AWS Configuration
	AWSRegion          string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSEndpointURL     string

	// SQS Configuration
	SQSQueueURL        string
	SQSDLQUrl          string
	SQSMaxMessages     int64
	SQSWaitTimeSeconds int64

	// DynamoDB Configuration
	DynamoDBTableName string
	DynamoDBEndpoint  string

	// Service Configuration
	ServicePort    string
	WorkerPoolSize int
	LogLevel       string
	SchemaPath     string
}

func Load() *Config {
	return &Config{
		// AWS Configuration
		AWSRegion:          getEnv("AWS_REGION", "us-east-1"),
		AWSAccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", "test"),
		AWSSecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", "test"),
		AWSEndpointURL:     getEnv("AWS_ENDPOINT_URL", "http://localhost:4566"),

		// SQS Configuration
		SQSQueueURL:        getEnv("SQS_QUEUE_URL", "http://localhost:4566/000000000000/event-queue"),
		SQSDLQUrl:          getEnv("SQS_DLQ_URL", "http://localhost:4566/000000000000/event-dlq"),
		SQSMaxMessages:     getEnvAsInt64("SQS_MAX_MESSAGES", 10),
		SQSWaitTimeSeconds: getEnvAsInt64("SQS_WAIT_TIME_SECONDS", 20),

		// DynamoDB Configuration
		DynamoDBTableName: getEnv("DYNAMODB_TABLE_NAME", "events"),
		DynamoDBEndpoint:  getEnv("DYNAMODB_ENDPOINT", "http://localhost:4566"),

		// Service Configuration
		ServicePort:    getEnv("SERVICE_PORT", "8080"),
		WorkerPoolSize: getEnvAsInt("WORKER_POOL_SIZE", 10),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		SchemaPath:     getEnv("SCHEMA_PATH", "../../schemas/event-schema.json"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}
