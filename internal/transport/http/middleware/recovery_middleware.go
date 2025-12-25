package middleware

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"

	"github.com/bravo68web/stasis/pkg/logger"
)

// RecoveryConfig holds configuration for the recovery middleware
type RecoveryConfig struct {
	// Logger is the logger instance to use
	Logger *logger.Logger

	// EnableStackTrace determines if stack traces should be logged
	EnableStackTrace bool

	// StackTraceSize is the maximum size of stack trace to capture
	StackTraceSize int
}

// DefaultRecoveryConfig returns a default recovery configuration
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		Logger:           nil, // Will use global logger
		EnableStackTrace: true,
		StackTraceSize:   4096,
	}
}

// RecoveryMiddleware returns a Gin middleware for panic recovery with logging
func RecoveryMiddleware() gin.HandlerFunc {
	return RecoveryMiddlewareWithConfig(DefaultRecoveryConfig())
}

// RecoveryMiddlewareWithConfig returns a panic recovery middleware with custom configuration
func RecoveryMiddlewareWithConfig(cfg *RecoveryConfig) gin.HandlerFunc {
	if cfg == nil {
		cfg = DefaultRecoveryConfig()
	}

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get logger
				log := cfg.Logger
				if log == nil {
					log = logger.Get()
				}

				// Get request ID if available
				requestID := GetRequestID(c)

				// Extract trace information if available
				traceID := ""
				spanID := ""
				span := trace.SpanFromContext(c.Request.Context())
				if span.SpanContext().IsValid() {
					traceID = span.SpanContext().TraceID().String()
					spanID = span.SpanContext().SpanID().String()
				}

				// Build log fields
				fields := []logger.Field{
					logger.Any("panic", err),
					logger.Method(c.Request.Method),
					logger.Path(c.Request.URL.Path),
					logger.Query(c.Request.URL.RawQuery),
					logger.ClientIP(c.ClientIP()),
					logger.UserAgent(c.Request.UserAgent()),
					logger.Time("recovered_at", time.Now()),
				}

				// Add request ID if available
				if requestID != "" {
					fields = append(fields, logger.RequestID(requestID))
				}

				// Add trace information if available
				if traceID != "" {
					fields = append(fields, logger.TraceID(traceID))
				}
				if spanID != "" {
					fields = append(fields, logger.SpanID(spanID))
				}

				// Add stack trace if enabled
				if cfg.EnableStackTrace {
					stack := debug.Stack()
					if len(stack) > cfg.StackTraceSize {
						stack = stack[:cfg.StackTraceSize]
					}
					fields = append(fields, logger.ByteString("stacktrace", stack))
				}

				// Log the panic
				log.Error("Panic recovered", fields...)

				// Check if connection is still alive
				if c.IsAborted() {
					return
				}

				// Set appropriate headers
				c.Header("Content-Type", "application/json")

				// Abort with error response
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":      "internal_server_error",
					"message":    "An unexpected error occurred",
					"request_id": requestID,
				})
			}
		}()

		c.Next()
	}
}

// RecoveryMiddlewareWithLogger returns a panic recovery middleware with a specific logger
func RecoveryMiddlewareWithLogger(log *logger.Logger) gin.HandlerFunc {
	return RecoveryMiddlewareWithConfig(&RecoveryConfig{
		Logger:           log,
		EnableStackTrace: true,
		StackTraceSize:   4096,
	})
}

// RecoveryMiddlewareNoStackTrace returns a panic recovery middleware without stack traces
// This is useful for production environments where you don't want to expose stack traces
func RecoveryMiddlewareNoStackTrace() gin.HandlerFunc {
	return RecoveryMiddlewareWithConfig(&RecoveryConfig{
		Logger:           nil,
		EnableStackTrace: false,
		StackTraceSize:   0,
	})
}

// CustomRecoveryMiddleware returns a panic recovery middleware with a custom error handler
func CustomRecoveryMiddleware(handler func(c *gin.Context, err interface{})) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				handler(c, err)
			}
		}()

		c.Next()
	}
}
