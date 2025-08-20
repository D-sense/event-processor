package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/google/uuid"

	"github.com/d-sense/event-processor/internal/config"
	awsPkg "github.com/d-sense/event-processor/pkg/aws"
	"github.com/d-sense/event-processor/pkg/models"
)

func main() {
	log.Println("Starting Event Producer...")

	// Load configuration
	cfg := config.Load()

	// Get producer rate from environment
	rate := 1 // Default: 1 event per second
	if rateStr := os.Getenv("PRODUCER_RATE"); rateStr != "" {
		if r, err := strconv.Atoi(rateStr); err == nil {
			rate = r
		}
	}

	// Create AWS session
	sess, err := awsPkg.NewSession(cfg)
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	// Create SQS client
	sqsClient := sqs.New(sess, &aws.Config{
		Endpoint: aws.String(cfg.AWSEndpointURL),
	})

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	log.Printf("Producing events at rate: %d events/second", rate)
	log.Printf("Target queue: %s", cfg.SQSQueueURL)

	ticker := time.NewTicker(time.Second / time.Duration(rate))
	defer ticker.Stop()

	eventCount := 0
	for range ticker.C {
		event := generateRandomEvent()
		if err := sendEvent(sqsClient, cfg.SQSQueueURL, event); err != nil {
			log.Printf("Failed to send event: %v", err)
		} else {
			eventCount++
			log.Printf("Sent event %d: %s (type: %s, client: %s)",
				eventCount, event.EventID, event.EventType, event.ClientID)
		}
	}
}

func generateRandomEvent() *models.Event {
	eventTypes := []models.EventType{
		models.EventTypeMonitoring,
		models.EventTypeUserAction,
		models.EventTypeTransaction,
		models.EventTypeIntegration,
	}

	clients := []string{
		"client-123",
		"client-456",
		"client-789",
		"client-test",
	}

	eventType := eventTypes[rand.Intn(len(eventTypes))]
	clientID := clients[rand.Intn(len(clients))]

	// Generate event-type specific payload
	payload := generatePayloadForType(eventType)

	return models.NewEvent(eventType, clientID, payload, "1.0")
}

func generatePayloadForType(eventType models.EventType) map[string]interface{} {
	switch eventType {
	case models.EventTypeMonitoring:
		severities := []string{"low", "medium", "high", "critical"}
		return map[string]interface{}{
			"severity":   severities[rand.Intn(len(severities))],
			"metric":     fmt.Sprintf("cpu.usage.%d", rand.Intn(100)),
			"value":      rand.Float64() * 100,
			"host":       fmt.Sprintf("server-%d", rand.Intn(10)),
			"datacenter": fmt.Sprintf("dc-%d", rand.Intn(3)),
		}

	case models.EventTypeUserAction:
		actions := []string{"login", "logout", "view_page", "click_button", "submit_form"}
		return map[string]interface{}{
			"userId":    fmt.Sprintf("user-%d", rand.Intn(1000)),
			"action":    actions[rand.Intn(len(actions))],
			"resource":  fmt.Sprintf("/api/v1/resource/%d", rand.Intn(100)),
			"userAgent": "Mozilla/5.0 (compatible; EventProducer/1.0)",
			"ipAddress": fmt.Sprintf("192.168.%d.%d", rand.Intn(255), rand.Intn(255)),
		}

	case models.EventTypeTransaction:
		currencies := []string{"USD", "EUR", "GBP", "JPY"}
		return map[string]interface{}{
			"transactionId": uuid.New().String(),
			"amount":        rand.Float64() * 10000,
			"currency":      currencies[rand.Intn(len(currencies))],
			"merchantId":    fmt.Sprintf("merchant-%d", rand.Intn(100)),
			"cardLast4":     fmt.Sprintf("%04d", rand.Intn(10000)),
			"status":        "pending",
		}

	case models.EventTypeIntegration:
		operations := []string{"sync", "import", "export", "transform"}
		sources := []string{"database", "api", "file", "stream"}
		targets := []string{"warehouse", "cache", "queue", "service"}
		return map[string]interface{}{
			"source":      sources[rand.Intn(len(sources))],
			"target":      targets[rand.Intn(len(targets))],
			"operation":   operations[rand.Intn(len(operations))],
			"recordCount": rand.Intn(10000),
			"batchId":     uuid.New().String(),
		}

	default:
		return map[string]interface{}{
			"message": "Generic event payload",
			"data":    rand.Intn(1000),
		}
	}
}

func sendEvent(sqsClient *sqs.SQS, queueURL string, event *models.Event) error {
	// Convert event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Send message to SQS
	_, err = sqsClient.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(string(eventJSON)),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
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
