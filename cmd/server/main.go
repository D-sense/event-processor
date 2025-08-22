package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/d-sense/event-processor/internal/config"
	"github.com/d-sense/event-processor/internal/consumer"
	"github.com/d-sense/event-processor/internal/health"
	"github.com/d-sense/event-processor/internal/persistence"
	"github.com/d-sense/event-processor/internal/processor"
	"github.com/d-sense/event-processor/internal/validator"
	"github.com/d-sense/event-processor/pkg/aws"
	"github.com/d-sense/event-processor/pkg/logger"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup logging using centralized logger package
	log := logger.New(cfg.LogLevel)

	// Create AWS config
	awsCfg, err := aws.NewSession(cfg)
	if err != nil {
		log.Fatalf("Failed to create AWS config: %v", err)
	}

	// Initialize all the components
	repo := persistence.NewDynamoDBRepository(awsCfg, cfg)
	eventValidator := validator.New(cfg.SchemaPath)
	eventProcessor := processor.New(repo, eventValidator, log)
	eventConsumer := consumer.NewSQSConsumer(awsCfg, cfg, eventProcessor, log)
	healthChecker := health.New(repo, log)

	// Initialize infrastructure (tables and queues) if they don't exist
	// TODO: this task should be handled by IoC.
	infraManager := persistence.NewInfrastructureManager(awsCfg, log)
	if err := infraManager.SetupInfrastructure(context.Background()); err != nil {
		log.WithError(err).Warn("Failed to setup infrastructure, continuing anyway")
	} else {
		log.Info("Infrastructure setup completed successfully")
	}

	// Start HTTP server
	go func() {
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			status := healthChecker.Check(r.Context())
			w.Header().Set("Content-Type", "application/json")
			if status.Healthy {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}

			// Convert status to JSON
			jsonData, err := json.Marshal(status)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"Failed to marshal health status"}`))
				return
			}
			w.Write(jsonData)
		})
		log.WithField("port", cfg.ServicePort).Info("Starting HTTP server")
		if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.ServicePort), nil); err != nil {
			log.WithError(err).Fatal("HTTP server failed")
		}
	}()

	// Start event consumer
	go func() {
		log.Info("Starting event consumer")
		if err := eventConsumer.Start(context.Background()); err != nil {
			log.WithError(err).Error("Event consumer error")
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := eventConsumer.Stop(ctx); err != nil {
		log.WithError(err).Error("Failed to stop event consumer gracefully")
	}

	log.Info("Shutdown complete")
}
