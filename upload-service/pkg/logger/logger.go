package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

// Initialize sets up the logger with proper configuration
func Initialize() {
	// Get log level from environment variable or default to info
	levelStr := strings.ToLower(os.Getenv("LOG_LEVEL"))
	if levelStr == "" {
		levelStr = "info"
	}

	// Parse the log level
	level, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		level = zerolog.InfoLevel
	}

	// Configure zerolog
	zerolog.SetGlobalLevel(level)

	// Determine if we're in a Kubernetes environment
	// If so, use JSON output; otherwise, use pretty console output for development
	output := determineOutput()

	// Set up the logger with service info
	hostname, _ := os.Hostname()
	log = zerolog.New(output).
		With().
		Timestamp().
		Str("service", "video-upload-service").
		Str("host", hostname).
		Logger()

	Info("Logger initialized", map[string]interface{}{
		"level": level.String(),
	})
}

// determineOutput returns the appropriate writer based on environment
func determineOutput() io.Writer {
	// Check if we're in Kubernetes or local dev
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return os.Stdout // Plain JSON for Kubernetes
	}

	// Pretty console output for local development
	return zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
}

// Debug logs a debug message
func Debug(msg string, fields map[string]interface{}) {
	event := log.Debug().Str("level", "debug")
	for key, val := range fields {
		event = addField(event, key, val)
	}
	event.Msg(msg)
}

// Info logs an info message
func Info(msg string, fields map[string]interface{}) {
	event := log.Info().Str("level", "info")
	for key, val := range fields {
		event = addField(event, key, val)
	}
	event.Msg(msg)
}

// Warn logs a warning message
func Warn(msg string, fields map[string]interface{}) {
	event := log.Warn().Str("level", "warn")
	for key, val := range fields {
		event = addField(event, key, val)
	}
	event.Msg(msg)
}

// Error logs an error message
func Error(msg string, err error, fields map[string]interface{}) {
	event := log.Error().Str("level", "error")
	if err != nil {
		event = event.Err(err)
	}
	for key, val := range fields {
		event = addField(event, key, val)
	}
	event.Msg(msg)
}

// Fatal logs a fatal error message and exits the program
func Fatal(msg string, err error, fields map[string]interface{}) {
	event := log.Fatal().Str("level", "fatal")
	if err != nil {
		event = event.Err(err)
	}
	for key, val := range fields {
		event = addField(event, key, val)
	}
	event.Msg(msg)
}

// addField adds a field of appropriate type to the event
func addField(event *zerolog.Event, key string, val interface{}) *zerolog.Event {
	switch v := val.(type) {
	case int:
		return event.Int(key, v)
	case int64:
		return event.Int64(key, v)
	case float64:
		return event.Float64(key, v)
	case string:
		return event.Str(key, v)
	case bool:
		return event.Bool(key, v)
	case error:
		return event.Err(v)
	default:
		return event.Interface(key, v)
	}
}

// WithCorrelationID creates a context logger with a correlation ID
func WithCorrelationID(correlationID string) *zerolog.Logger {
	contextLogger := log.With().Str("correlation_id", correlationID).Logger()
	return &contextLogger
}
