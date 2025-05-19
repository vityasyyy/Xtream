package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger middleware for adding request logging with correlation IDs
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Create correlation ID
		correlationID := uuid.New().String()
		c.Set("correlation_id", correlationID)
		c.Header("X-Correlation-ID", correlationID)

		// Create a request-specific logger
		reqLogger := log.With().
			Str("correlation_id", correlationID).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("client_ip", c.ClientIP()).
			Logger()

		// Add logger to context
		c.Set("logger", &reqLogger)

		// Process request
		c.Next()

		// Calculate request time
		duration := time.Since(start)

		// Log request details after completion
		status := c.Writer.Status()
		logEvent := reqLogger.Info()

		// Determine log level based on status code
		if status >= 400 {
			logEvent = reqLogger.Error()
		}

		// Create log entry
		logEvent.
			Int("status_code", status).
			Dur("duration", duration).
			Int("size", c.Writer.Size()).
			Msg(fmt.Sprintf("%s %s - %d (%s)",
				c.Request.Method,
				c.Request.URL.Path,
				status,
				duration))
	}
}

// GetLogger extracts the request-specific logger from gin context
func GetLogger(c *gin.Context) *zerolog.Logger {
	loggerInterface, exists := c.Get("logger")
	if !exists {
		return &log.Logger // Return global logger if not found
	}

	logger, ok := loggerInterface.(*zerolog.Logger)
	if !ok {
		return &log.Logger // Return global logger if type assertion fails
	}

	return logger
}

// GetCorrelationID extracts the correlation ID from gin context
func GetCorrelationID(c *gin.Context) string {
	correlationID, exists := c.Get("correlation_id")
	if !exists {
		return "unknown"
	}

	id, ok := correlationID.(string)
	if !ok {
		return "unknown"
	}

	return id
}
