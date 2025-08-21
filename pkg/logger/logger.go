package logger

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// New creates a new configured logger instance
func New(level string) *logrus.Logger {
	logger := logrus.New()

	// Set log level
	switch strings.ToLower(level) {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn", "warning":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logger.SetLevel(logrus.FatalLevel)
	case "panic":
		logger.SetLevel(logrus.PanicLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// Set JSON formatter for structured logging
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
			logrus.FieldKeyFunc:  "function",
		},
	})

	// Set output to stdout
	logger.SetOutput(os.Stdout)

	// Add default fields
	logger.WithFields(logrus.Fields{
		"service": "event-processor",
		"version": "1.0.0",
	})

	return logger
}

// WithCorrelationID adds correlation ID to logger context
func WithCorrelationID(logger *logrus.Logger, correlationID string) *logrus.Entry {
	return logger.WithField("correlation_id", correlationID)
}

// WithComponent adds component name to logger context
func WithComponent(logger *logrus.Logger, component string) *logrus.Entry {
	return logger.WithField("component", component)
}

// WithEventContext adds event-specific context to logger
func WithEventContext(logger *logrus.Logger, eventID, eventType, clientID string) *logrus.Entry {
	return logger.WithFields(logrus.Fields{
		"event_id":   eventID,
		"event_type": eventType,
		"client_id":  clientID,
	})
}

// WithRequestContext adds HTTP request context to logger
func WithRequestContext(logger *logrus.Logger, method, path, userAgent string) *logrus.Entry {
	return logger.WithFields(logrus.Fields{
		"http_method": method,
		"http_path":   path,
		"user_agent":  userAgent,
	})
}

// WithError adds error context to logger
func WithError(logger *logrus.Logger, err error) *logrus.Entry {
	return logger.WithField("error", err.Error())
}

// WithFields adds multiple fields to logger context
func WithFields(logger *logrus.Logger, fields map[string]interface{}) *logrus.Entry {
	return logger.WithFields(logrus.Fields(fields))
}

// WithDuration adds duration field to logger context
func WithDuration(logger *logrus.Logger, duration string) *logrus.Entry {
	return logger.WithField("duration", duration)
}

// WithCount adds count field to logger context
func WithCount(logger *logrus.Logger, count int) *logrus.Entry {
	return logger.WithField("count", count)
}

// WithStatus adds status field to logger context
func WithStatus(logger *logrus.Logger, status string) *logrus.Entry {
	return logger.WithField("status", status)
}
