package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

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
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found")
	}

	// Initialize configuration
	cfg := config.Load()

	// Initialize logger
	log := logger.New(cfg.LogLevel)

	// Initialize AWS session
	sess, err := aws.NewSession(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to create AWS session")
	}

	// Initialize components
	repo := persistence.NewDynamoDBRepository(sess, cfg)
	eventValidator := validator.New(cfg.SchemaPath)
	eventProcessor := processor.New(repo, eventValidator, log)
	eventConsumer := consumer.NewSQSConsumer(sess, cfg, eventProcessor, log)
	healthChecker := health.New(repo, log)

	// Setup HTTP server
	router := setupRouter(healthChecker, eventValidator, log)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.ServicePort),
		Handler: router,
	}

	// Start HTTP server
	go func() {
		log.WithField("port", cfg.ServicePort).Info("Starting HTTP server")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	// Start event consumer
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		log.Info("Starting event consumer")
		if err := eventConsumer.Start(ctx); err != nil {
			log.WithError(err).Error("Event consumer error")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown
	cancel() // Cancel consumer context

	// Shutdown HTTP server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Error("Server forced to shutdown")
	}

	log.Info("Server exited")
}

func setupRouter(healthChecker *health.HealthChecker, validator *validator.Validator, log *logrus.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()

		log.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"duration":   time.Since(start),
			"user_agent": c.Request.UserAgent(),
		}).Info("HTTP request")
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		status := healthChecker.Check(c.Request.Context())
		if status.Healthy {
			c.JSON(http.StatusOK, status)
		} else {
			c.JSON(http.StatusServiceUnavailable, status)
		}
	})

	// Metrics endpoint
	router.GET("/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "event-processor",
			"status":  "running",
		})
	})

	// Event validation endpoint for testing
	router.POST("/validate", func(c *gin.Context) {
		var payload interface{}
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := validator.ValidateEvent(payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"valid": false,
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"valid": true})
	})

	return router
}
