package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test data structures
type loadConfigTestCase struct {
	name           string
	envVars        map[string]string
	expectedConfig *Config
	description    string
}

type getEnvTestCase struct {
	name           string
	key            string
	defaultValue   string
	envValue       string
	expectedResult string
	description    string
}

type getEnvAsIntTestCase struct {
	name           string
	key            string
	defaultValue   int
	envValue       string
	expectedResult int
	description    string
}

type getEnvAsInt64TestCase struct {
	name           string
	key            string
	defaultValue   int64
	envValue       string
	expectedResult int64
	description    string
}

// TestLoad tests the Load function
func TestLoad(t *testing.T) {
	tests := []loadConfigTestCase{
		{
			name:    "Default Configuration - No Environment Variables",
			envVars: map[string]string{},
			expectedConfig: &Config{
				AWSRegion:          "us-east-1",
				AWSAccessKeyID:     "test",
				AWSSecretAccessKey: "test",
				AWSEndpointURL:     "http://localhost:4566",
				SQSQueueURL:        "http://localhost:4566/000000000000/event-queue",
				SQSDLQUrl:          "http://localhost:4566/000000000000/event-dlq",
				SQSMaxMessages:     10,
				SQSWaitTimeSeconds: 20,
				DynamoDBTableName:  "events",
				DynamoDBEndpoint:   "http://localhost:4566",
				ServicePort:        "8080",
				WorkerPoolSize:     10,
				LogLevel:           "info",
				SchemaPath:         "../../schemas/event-schema.json",
			},
			description: "Should load default configuration when no environment variables are set",
		},
		{
			name: "Custom AWS Configuration",
			envVars: map[string]string{
				"AWS_REGION":            "eu-west-1",
				"AWS_ACCESS_KEY_ID":     "custom-key",
				"AWS_SECRET_ACCESS_KEY": "custom-secret",
				"AWS_ENDPOINT_URL":      "https://custom-endpoint.com",
			},
			expectedConfig: &Config{
				AWSRegion:          "eu-west-1",
				AWSAccessKeyID:     "custom-key",
				AWSSecretAccessKey: "custom-secret",
				AWSEndpointURL:     "https://custom-endpoint.com",
				SQSQueueURL:        "http://localhost:4566/000000000000/event-queue",
				SQSDLQUrl:          "http://localhost:4566/000000000000/event-dlq",
				SQSMaxMessages:     10,
				SQSWaitTimeSeconds: 20,
				DynamoDBTableName:  "events",
				DynamoDBEndpoint:   "https://custom-endpoint.com", // Should use AWS_ENDPOINT_URL
				ServicePort:        "8080",
				WorkerPoolSize:     10,
				LogLevel:           "info",
				SchemaPath:         "../../schemas/event-schema.json",
			},
			description: "Should override AWS configuration with environment variables",
		},
		{
			name: "Custom SQS Configuration",
			envVars: map[string]string{
				"SQS_QUEUE_URL":         "https://sqs.custom.com/queue",
				"SQS_DLQ_URL":           "https://sqs.custom.com/dlq",
				"SQS_MAX_MESSAGES":      "25",
				"SQS_WAIT_TIME_SECONDS": "30",
			},
			expectedConfig: &Config{
				AWSRegion:          "us-east-1",
				AWSAccessKeyID:     "test",
				AWSSecretAccessKey: "test",
				AWSEndpointURL:     "http://localhost:4566",
				SQSQueueURL:        "https://sqs.custom.com/queue",
				SQSDLQUrl:          "https://sqs.custom.com/dlq",
				SQSMaxMessages:     25,
				SQSWaitTimeSeconds: 30,
				DynamoDBTableName:  "events",
				DynamoDBEndpoint:   "http://localhost:4566",
				ServicePort:        "8080",
				WorkerPoolSize:     10,
				LogLevel:           "info",
				SchemaPath:         "../../schemas/event-schema.json",
			},
			description: "Should override SQS configuration with environment variables",
		},
		{
			name: "Custom DynamoDB Configuration",
			envVars: map[string]string{
				"DYNAMODB_TABLE_NAME": "custom-events-table",
			},
			expectedConfig: &Config{
				AWSRegion:          "us-east-1",
				AWSAccessKeyID:     "test",
				AWSSecretAccessKey: "test",
				AWSEndpointURL:     "http://localhost:4566",
				SQSQueueURL:        "http://localhost:4566/000000000000/event-queue",
				SQSDLQUrl:          "http://localhost:4566/000000000000/event-dlq",
				SQSMaxMessages:     10,
				SQSWaitTimeSeconds: 20,
				DynamoDBTableName:  "custom-events-table",
				DynamoDBEndpoint:   "http://localhost:4566",
				ServicePort:        "8080",
				WorkerPoolSize:     10,
				LogLevel:           "info",
				SchemaPath:         "../../schemas/event-schema.json",
			},
			description: "Should override DynamoDB table name with environment variable",
		},
		{
			name: "Custom Service Configuration",
			envVars: map[string]string{
				"SERVICE_PORT":     "9090",
				"WORKER_POOL_SIZE": "20",
				"LOG_LEVEL":        "debug",
				"SCHEMA_PATH":      "/custom/schema/path.json",
			},
			expectedConfig: &Config{
				AWSRegion:          "us-east-1",
				AWSAccessKeyID:     "test",
				AWSSecretAccessKey: "test",
				AWSEndpointURL:     "http://localhost:4566",
				SQSQueueURL:        "http://localhost:4566/000000000000/event-queue",
				SQSDLQUrl:          "http://localhost:4566/000000000000/event-dlq",
				SQSMaxMessages:     10,
				SQSWaitTimeSeconds: 20,
				DynamoDBTableName:  "events",
				DynamoDBEndpoint:   "http://localhost:4566",
				ServicePort:        "9090",
				WorkerPoolSize:     20,
				LogLevel:           "debug",
				SchemaPath:         "/custom/schema/path.json",
			},
			description: "Should override service configuration with environment variables",
		},
		{
			name: "Complete Custom Configuration",
			envVars: map[string]string{
				"AWS_REGION":            "ap-southeast-1",
				"AWS_ACCESS_KEY_ID":     "prod-key",
				"AWS_SECRET_ACCESS_KEY": "prod-secret",
				"AWS_ENDPOINT_URL":      "https://prod-endpoint.aws.com",
				"SQS_QUEUE_URL":         "https://prod-sqs.aws.com/events",
				"SQS_DLQ_URL":           "https://prod-sqs.aws.com/events-dlq",
				"SQS_MAX_MESSAGES":      "50",
				"SQS_WAIT_TIME_SECONDS": "60",
				"DYNAMODB_TABLE_NAME":   "prod-events",
				"SERVICE_PORT":          "8443",
				"WORKER_POOL_SIZE":      "50",
				"LOG_LEVEL":             "warn",
				"SCHEMA_PATH":           "/etc/event-processor/schemas/event-schema.json",
			},
			expectedConfig: &Config{
				AWSRegion:          "ap-southeast-1",
				AWSAccessKeyID:     "prod-key",
				AWSSecretAccessKey: "prod-secret",
				AWSEndpointURL:     "https://prod-endpoint.aws.com",
				SQSQueueURL:        "https://prod-sqs.aws.com/events",
				SQSDLQUrl:          "https://prod-sqs.aws.com/events-dlq",
				SQSMaxMessages:     50,
				SQSWaitTimeSeconds: 60,
				DynamoDBTableName:  "prod-events",
				DynamoDBEndpoint:   "https://prod-endpoint.aws.com",
				ServicePort:        "8443",
				WorkerPoolSize:     50,
				LogLevel:           "warn",
				SchemaPath:         "/etc/event-processor/schemas/event-schema.json",
			},
			description: "Should override all configuration with environment variables",
		},
		{
			name: "Mixed Configuration - Some Custom, Some Default",
			envVars: map[string]string{
				"AWS_REGION":       "ca-central-1",
				"SQS_QUEUE_URL":    "https://mixed-sqs.com/queue",
				"WORKER_POOL_SIZE": "15",
				"LOG_LEVEL":        "error",
			},
			expectedConfig: &Config{
				AWSRegion:          "ca-central-1",
				AWSAccessKeyID:     "test",
				AWSSecretAccessKey: "test",
				AWSEndpointURL:     "http://localhost:4566",
				SQSQueueURL:        "https://mixed-sqs.com/queue",
				SQSDLQUrl:          "http://localhost:4566/000000000000/event-dlq",
				SQSMaxMessages:     10,
				SQSWaitTimeSeconds: 20,
				DynamoDBTableName:  "events",
				DynamoDBEndpoint:   "http://localhost:4566",
				ServicePort:        "8080",
				WorkerPoolSize:     15,
				LogLevel:           "error",
				SchemaPath:         "../../schemas/event-schema.json",
			},
			description: "Should mix custom and default configuration values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variables for this test
			setupTestEnvironment(tt.envVars)
			defer cleanupTestEnvironment(tt.envVars)

			// Execute test
			result := Load()

			// Assertions
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedConfig.AWSRegion, result.AWSRegion)
			assert.Equal(t, tt.expectedConfig.AWSAccessKeyID, result.AWSAccessKeyID)
			assert.Equal(t, tt.expectedConfig.AWSSecretAccessKey, result.AWSSecretAccessKey)
			assert.Equal(t, tt.expectedConfig.AWSEndpointURL, result.AWSEndpointURL)
			assert.Equal(t, tt.expectedConfig.SQSQueueURL, result.SQSQueueURL)
			assert.Equal(t, tt.expectedConfig.SQSDLQUrl, result.SQSDLQUrl)
			assert.Equal(t, tt.expectedConfig.SQSMaxMessages, result.SQSMaxMessages)
			assert.Equal(t, tt.expectedConfig.SQSWaitTimeSeconds, result.SQSWaitTimeSeconds)
			assert.Equal(t, tt.expectedConfig.DynamoDBTableName, result.DynamoDBTableName)
			assert.Equal(t, tt.expectedConfig.DynamoDBEndpoint, result.DynamoDBEndpoint)
			assert.Equal(t, tt.expectedConfig.ServicePort, result.ServicePort)
			assert.Equal(t, tt.expectedConfig.WorkerPoolSize, result.WorkerPoolSize)
			assert.Equal(t, tt.expectedConfig.LogLevel, result.LogLevel)
			assert.Equal(t, tt.expectedConfig.SchemaPath, result.SchemaPath)
		})
	}
}

// TestGetEnv tests the getEnv function
func TestGetEnv(t *testing.T) {
	tests := []getEnvTestCase{
		{
			name:           "Environment Variable Set",
			key:            "TEST_KEY",
			defaultValue:   "default-value",
			envValue:       "custom-value",
			expectedResult: "custom-value",
			description:    "Should return environment variable value when set",
		},
		{
			name:           "Environment Variable Not Set",
			key:            "MISSING_KEY",
			defaultValue:   "default-value",
			envValue:       "",
			expectedResult: "default-value",
			description:    "Should return default value when environment variable is not set",
		},
		{
			name:           "Environment Variable Empty String",
			key:            "EMPTY_KEY",
			defaultValue:   "default-value",
			envValue:       "",
			expectedResult: "default-value",
			description:    "Should return default value when environment variable is empty string",
		},
		{
			name:           "Environment Variable with Spaces",
			key:            "SPACES_KEY",
			defaultValue:   "default-value",
			envValue:       "  value with spaces  ",
			expectedResult: "  value with spaces  ",
			description:    "Should preserve spaces in environment variable value",
		},
		{
			name:           "Environment Variable with Special Characters",
			key:            "SPECIAL_KEY",
			defaultValue:   "default-value",
			envValue:       "!@#$%^&*()_+-=[]{}|;':\",./<>?",
			expectedResult: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
			description:    "Should handle special characters in environment variable value",
		},
		{
			name:           "Environment Variable with Newlines",
			key:            "NEWLINE_KEY",
			defaultValue:   "default-value",
			envValue:       "line1\nline2\nline3",
			expectedResult: "line1\nline2\nline3",
			description:    "Should handle newlines in environment variable value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variable for this test
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			// Execute test
			result := getEnv(tt.key, tt.defaultValue)

			// Assertions
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// TestGetEnvAsInt tests the getEnvAsInt function
func TestGetEnvAsInt(t *testing.T) {
	tests := []getEnvAsIntTestCase{
		{
			name:           "Valid Integer Environment Variable",
			key:            "INT_KEY",
			defaultValue:   42,
			envValue:       "123",
			expectedResult: 123,
			description:    "Should return parsed integer when environment variable contains valid integer",
		},
		{
			name:           "Environment Variable Not Set",
			key:            "MISSING_INT_KEY",
			defaultValue:   42,
			envValue:       "",
			expectedResult: 42,
			description:    "Should return default value when environment variable is not set",
		},
		{
			name:           "Invalid Integer Environment Variable",
			key:            "INVALID_INT_KEY",
			defaultValue:   42,
			envValue:       "not-a-number",
			expectedResult: 42,
			description:    "Should return default value when environment variable contains invalid integer",
		},
		{
			name:           "Zero Integer Environment Variable",
			key:            "ZERO_INT_KEY",
			defaultValue:   42,
			envValue:       "0",
			expectedResult: 0,
			description:    "Should return zero when environment variable contains '0'",
		},
		{
			name:           "Negative Integer Environment Variable",
			key:            "NEGATIVE_INT_KEY",
			defaultValue:   42,
			envValue:       "-123",
			expectedResult: -123,
			description:    "Should return negative integer when environment variable contains valid negative integer",
		},
		{
			name:           "Large Integer Environment Variable",
			key:            "LARGE_INT_KEY",
			defaultValue:   42,
			envValue:       "2147483647", // Max int32
			expectedResult: 2147483647,
			description:    "Should handle large integer values",
		},
		{
			name:           "Float String Environment Variable",
			key:            "FLOAT_INT_KEY",
			defaultValue:   42,
			envValue:       "123.45",
			expectedResult: 42,
			description:    "Should return default value when environment variable contains float string",
		},
		{
			name:           "Empty String Environment Variable",
			key:            "EMPTY_INT_KEY",
			defaultValue:   42,
			envValue:       "",
			expectedResult: 42,
			description:    "Should return default value when environment variable is empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variable for this test
			if tt.envValue != "" {
				err := os.Setenv(tt.key, tt.envValue)
				if err != nil {
					return
				}
				defer func(key string) {
					err := os.Unsetenv(key)
					if err != nil {
						return
					}
				}(tt.key)
			}

			// Execute test
			result := getEnvAsInt(tt.key, tt.defaultValue)

			// Assertions
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// TestGetEnvAsInt64 tests the getEnvAsInt64 function
func TestGetEnvAsInt64(t *testing.T) {
	tests := []getEnvAsInt64TestCase{
		{
			name:           "Valid Int64 Environment Variable",
			key:            "INT64_KEY",
			defaultValue:   42,
			envValue:       "123",
			expectedResult: 123,
			description:    "Should return parsed int64 when environment variable contains valid integer",
		},
		{
			name:           "Environment Variable Not Set",
			key:            "MISSING_INT64_KEY",
			defaultValue:   42,
			envValue:       "",
			expectedResult: 42,
			description:    "Should return default value when environment variable is not set",
		},
		{
			name:           "Invalid Int64 Environment Variable",
			key:            "INVALID_INT64_KEY",
			defaultValue:   42,
			envValue:       "not-a-number",
			expectedResult: 42,
			description:    "Should return default value when environment variable contains invalid integer",
		},
		{
			name:           "Zero Int64 Environment Variable",
			key:            "ZERO_INT64_KEY",
			defaultValue:   42,
			envValue:       "0",
			expectedResult: 0,
			description:    "Should return zero when environment variable contains '0'",
		},
		{
			name:           "Negative Int64 Environment Variable",
			key:            "NEGATIVE_INT64_KEY",
			defaultValue:   42,
			envValue:       "-123",
			expectedResult: -123,
			description:    "Should return negative int64 when environment variable contains valid negative integer",
		},
		{
			name:           "Large Int64 Environment Variable",
			key:            "LARGE_INT64_KEY",
			defaultValue:   42,
			envValue:       "9223372036854775807", // Max int64
			expectedResult: 9223372036854775807,
			description:    "Should handle large int64 values",
		},
		{
			name:           "Float String Int64 Environment Variable",
			key:            "FLOAT_INT64_KEY",
			defaultValue:   42,
			envValue:       "123.45",
			expectedResult: 42,
			description:    "Should return default value when environment variable contains float string",
		},
		{
			name:           "Empty String Int64 Environment Variable",
			key:            "EMPTY_INT64_KEY",
			defaultValue:   42,
			envValue:       "",
			expectedResult: 42,
			description:    "Should return default value when environment variable is empty string",
		},
		{
			name:           "Very Large Int64 Environment Variable",
			key:            "VERY_LARGE_INT64_KEY",
			defaultValue:   42,
			envValue:       "1000000000000000000",
			expectedResult: 1000000000000000000,
			description:    "Should handle very large int64 values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variable for this test
			if tt.envValue != "" {
				err := os.Setenv(tt.key, tt.envValue)
				if err != nil {
					return
				}
				defer func(key string) {
					err := os.Unsetenv(key)
					if err != nil {
						return
					}
				}(tt.key)
			}

			// Execute test
			result := getEnvAsInt64(tt.key, tt.defaultValue)

			// Assertions
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// TestConfigStruct tests the Config struct fields
func TestConfigStruct(t *testing.T) {
	t.Run("Config Struct Fields", func(t *testing.T) {
		// Create a config instance
		config := &Config{
			AWSRegion:          "test-region",
			AWSAccessKeyID:     "test-key",
			AWSSecretAccessKey: "test-secret",
			AWSEndpointURL:     "http://test-endpoint.com",
			SQSQueueURL:        "http://test-sqs.com/queue",
			SQSDLQUrl:          "http://test-sqs.com/dlq",
			SQSMaxMessages:     25,
			SQSWaitTimeSeconds: 30,
			DynamoDBTableName:  "test-table",
			DynamoDBEndpoint:   "http://test-dynamodb.com",
			ServicePort:        "9090",
			WorkerPoolSize:     15,
			LogLevel:           "debug",
			SchemaPath:         "/test/schema.json",
		}

		// Assertions
		assert.Equal(t, "test-region", config.AWSRegion)
		assert.Equal(t, "test-key", config.AWSAccessKeyID)
		assert.Equal(t, "test-secret", config.AWSSecretAccessKey)
		assert.Equal(t, "http://test-endpoint.com", config.AWSEndpointURL)
		assert.Equal(t, "http://test-sqs.com/queue", config.SQSQueueURL)
		assert.Equal(t, "http://test-sqs.com/dlq", config.SQSDLQUrl)
		assert.Equal(t, int64(25), config.SQSMaxMessages)
		assert.Equal(t, int64(30), config.SQSWaitTimeSeconds)
		assert.Equal(t, "test-table", config.DynamoDBTableName)
		assert.Equal(t, "http://test-dynamodb.com", config.DynamoDBEndpoint)
		assert.Equal(t, "9090", config.ServicePort)
		assert.Equal(t, 15, config.WorkerPoolSize)
		assert.Equal(t, "debug", config.LogLevel)
		assert.Equal(t, "/test/schema.json", config.SchemaPath)
	})
}

// Helper functions for test setup and cleanup

func setupTestEnvironment(envVars map[string]string) {
	for key, value := range envVars {
		err := os.Setenv(key, value)
		if err != nil {
			return
		}
	}
}

func cleanupTestEnvironment(envVars map[string]string) {
	for key := range envVars {
		err := os.Unsetenv(key)
		if err != nil {
			return
		}
	}
}
