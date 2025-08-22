package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/d-sense/event-processor/pkg/models"
)

func main() {
	// Get configuration from environment variables
	endpoint := getEnv("AWS_ENDPOINT_URL", "http://localhost:4566")
	region := getEnv("AWS_REGION", "us-east-1")
	queueURL := getEnv("SQS_QUEUE_URL", "http://localhost:4566/000000000000/event-queue")

	log.Printf("Using AWS endpoint: %s", endpoint)
	log.Printf("Using queue URL: %s", queueURL)

	// Create AWS config
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     "test",
				SecretAccessKey: "test",
			},
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create AWS config: %v", err)
	}

	// Create SQS client
	sqsClient := sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Generate and send events continuously
	eventCounter := 0
	log.Println("Starting continuous event generation...")
	log.Println("Press Ctrl+C to stop gracefully")

	for {
		select {
		case <-sigChan:
			log.Println("Received shutdown signal, stopping gracefully...")
			log.Printf("Total events sent: %d", eventCounter)
			return
		default:
			eventCounter++
			event := generateRandomEvent()

			// Debug: Print the event before sending (less verbose for continuous mode)
			log.Printf("Generated event #%d: %s (Type: %s, Client: %s)",
				eventCounter, event.EventID, event.EventType, event.ClientID)

			if err := sendEvent(sqsClient, queueURL, event); err != nil {
				log.Printf("Failed to send event: %v", err)
				continue
			}

			log.Printf("✅ Sent event #%d: %s (Type: %s, Client: %s)",
				eventCounter, event.EventID, event.EventType, event.ClientID)

			// Wait between events (configurable interval)
			time.Sleep(2 * time.Second)
		}
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// generateRandomEvent creates a random event for testing
func generateRandomEvent() *models.Event {
	eventTypes := []models.EventType{
		models.EventTypeMonitoring,
		models.EventTypeUserAction,
		models.EventTypeTransaction,
		models.EventTypeIntegration,
	}

	clients := []string{"client-001", "client-002", "client-003"}

	// Generate payload based on event type first
	eventType := eventTypes[rand.Intn(len(eventTypes))]
	payload := generatePayloadForType(eventType)

	event := &models.Event{
		EventID:   uuid.NewString(),
		EventType: eventType,
		ClientID:  clients[rand.Intn(len(clients))],
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		Payload:   payload, // Set payload only once
	}

	return event
}

// generatePayloadForType creates appropriate payload for different event types
func generatePayloadForType(eventType models.EventType) map[string]interface{} {
	switch eventType {
	case models.EventTypeMonitoring:
		return map[string]interface{}{
			"severity": "medium",
			"metric":   "cpu.usage",
			"value":    rand.Float64() * 100,
			"host":     "test-server",
		}
	case models.EventTypeUserAction:
		return map[string]interface{}{
			"userId":    uuid.NewString(),
			"action":    "login",
			"resource":  "/api/auth",
			"userAgent": "test-client/1.0",
		}
	case models.EventTypeTransaction:
		return map[string]interface{}{
			"transactionId": uuid.NewString(),
			"amount":        rand.Float64() * 1000,
			"currency":      "USD",
			"status":        "pending",
		}
	case models.EventTypeIntegration:
		return map[string]interface{}{
			"source":       "test-api",
			"target":       "test-db",
			"operation":    "sync",
			"record_count": rand.Intn(1000),
		}
	default:
		return map[string]interface{}{
			"message": "Unknown event type",
		}
	}
}

// sendEvent sends an event to SQS
func sendEvent(sqsClient *sqs.Client, queueURL string, event *models.Event) error {
	// Convert event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Debug: Print the JSON being sent
	log.Printf("Sending JSON payload: %s", string(eventJSON))

	// Send message to SQS
	_, err = sqsClient.SendMessage(context.Background(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(string(eventJSON)),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"EventType": {
				DataType:    aws.String("String"),
				StringValue: aws.String(string(event.EventType)),
			},
			"ClientID": {
				DataType:    aws.String("String"),
				StringValue: aws.String(event.ClientID),
			},
		},
	})

	return err
}
