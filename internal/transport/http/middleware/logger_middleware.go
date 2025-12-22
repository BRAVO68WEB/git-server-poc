package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"

	"github.com/bravo68web/stasis/pkg/logger"
)

// LoggerConfig holds configuration for the logging middleware
type LoggerConfig struct {
	// Logger is the logger instance to use
	Logger *logger.Logger

	// SkipPaths are paths that should not be logged
	SkipPaths []string

	// SkipPathPrefixes are path prefixes that should not be logged
	SkipPathPrefixes []string

	// LogRequestBody determines if request body should be logged
	LogRequestBody bool

	// LogResponseBody determines if response body should be logged
	LogResponseBody bool

	// MaxBodyLogSize is the maximum size of body to log (in bytes)
	MaxBodyLogSize int

	// TraceIDHeader is the header name for trace ID (for external trace propagation)
	TraceIDHeader string

	// RequestIDHeader is the header name for request ID
	RequestIDHeader string

	// IncludeHeaders determines if request headers should be logged
	IncludeHeaders bool

	// SensitiveHeaders are headers that should be redacted
	SensitiveHeaders []string
}

// DefaultLoggerConfig returns a default middleware configuration
func DefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		Logger:           nil, // Will use global logger
		SkipPaths:        []string{"/health", "/healthz", "/ready", "/readyz", "/metrics"},
		SkipPathPrefixes: []string{},
		LogRequestBody:   false,
		LogResponseBody:  false,
		MaxBodyLogSize:   1024, // 1KB
		TraceIDHeader:    "X-Trace-ID",
		RequestIDHeader:  "X-Request-ID",
		IncludeHeaders:   false,
		SensitiveHeaders: []string{"Authorization", "Cookie", "X-API-Key", "X-Auth-Token"},
	}
}

// responseBodyWriter wraps gin.ResponseWriter to capture response body
type responseBodyWriter struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseBodyWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseBodyWriter) Status() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}
	return w.statusCode
}

// LoggerMiddleware returns a Gin middleware for logging HTTP requests
func LoggerMiddleware() gin.HandlerFunc {
	return LoggerMiddlewareWithConfig(DefaultLoggerConfig())
}

// LoggerMiddlewareWithConfig returns a Gin middleware with custom configuration
func LoggerMiddlewareWithConfig(cfg *LoggerConfig) gin.HandlerFunc {
	if cfg == nil {
		cfg = DefaultLoggerConfig()
	}

	// Build skip paths map for O(1) lookup
	skipPaths := make(map[string]struct{}, len(cfg.SkipPaths))
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = struct{}{}
	}

	// Build sensitive headers map for O(1) lookup
	sensitiveHeaders := make(map[string]struct{}, len(cfg.SensitiveHeaders))
	for _, header := range cfg.SensitiveHeaders {
		sensitiveHeaders[header] = struct{}{}
	}

	return func(c *gin.Context) {
		// Check if path should be skipped
		path := c.Request.URL.Path
		if _, ok := skipPaths[path]; ok {
			c.Next()
			return
		}

		// Check if path matches any skip prefix
		for _, prefix := range cfg.SkipPathPrefixes {
			if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
				c.Next()
				return
			}
		}

		// Get logger
		log := cfg.Logger
		if log == nil {
			log = logger.Get()
		}

		// Start timer
		start := time.Now()

		// Extract request ID
		requestID := c.GetHeader(cfg.RequestIDHeader)
		if requestID == "" {
			requestID = generateRequestID()
			c.Header(cfg.RequestIDHeader, requestID)
		}

		// Extract trace and span IDs from OTEL context
		traceID := c.GetHeader(cfg.TraceIDHeader)
		spanID := ""

		span := trace.SpanFromContext(c.Request.Context())
		if span.SpanContext().IsValid() {
			traceID = span.SpanContext().TraceID().String()
			spanID = span.SpanContext().SpanID().String()
		}

		// Set trace ID in response header if available
		if traceID != "" {
			c.Header(cfg.TraceIDHeader, traceID)
		}

		// Store request ID in context for downstream use
		c.Set("request_id", requestID)
		if traceID != "" {
			c.Set("trace_id", traceID)
		}
		if spanID != "" {
			c.Set("span_id", spanID)
		}

		// Read request body if configured
		var requestBody string
		if cfg.LogRequestBody && c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				// Restore the body so it can be read again
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				// Truncate if too large
				if len(bodyBytes) > cfg.MaxBodyLogSize {
					requestBody = string(bodyBytes[:cfg.MaxBodyLogSize]) + "...(truncated)"
				} else {
					requestBody = string(bodyBytes)
				}
			}
		}

		// Create response writer wrapper if logging response body
		var rbw *responseBodyWriter
		if cfg.LogResponseBody {
			rbw = &responseBodyWriter{
				ResponseWriter: c.Writer,
				body:           bytes.NewBuffer(nil),
			}
			c.Writer = rbw
		}

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		statusCode := c.Writer.Status()
		if rbw != nil {
			statusCode = rbw.Status()
		}

		// Build log fields
		fields := []logger.Field{
			logger.RequestID(requestID),
			logger.Method(c.Request.Method),
			logger.Path(path),
			logger.Query(c.Request.URL.RawQuery),
			logger.StatusCode(statusCode),
			logger.Latency(latency),
			logger.String("latency_human", latency.String()),
			logger.ClientIP(c.ClientIP()),
			logger.UserAgent(c.Request.UserAgent()),
			logger.BodySize(c.Writer.Size()),
			logger.Protocol(c.Request.Proto),
		}

		// Add trace information
		if traceID != "" {
			fields = append(fields, logger.TraceID(traceID))
		}
		if spanID != "" {
			fields = append(fields, logger.SpanID(spanID))
		}

		// Add referer if present
		if referer := c.Request.Referer(); referer != "" {
			fields = append(fields, logger.Referer(referer))
		}

		// Add request body if captured
		if requestBody != "" {
			fields = append(fields, logger.String("request_body", requestBody))
		}

		// Add response body if captured
		if cfg.LogResponseBody && rbw != nil && rbw.body.Len() > 0 {
			responseBody := rbw.body.String()
			if len(responseBody) > cfg.MaxBodyLogSize {
				responseBody = responseBody[:cfg.MaxBodyLogSize] + "...(truncated)"
			}
			fields = append(fields, logger.String("response_body", responseBody))
		}

		// Add headers if configured
		if cfg.IncludeHeaders {
			headers := make(map[string]string)
			for key, values := range c.Request.Header {
				if _, sensitive := sensitiveHeaders[key]; sensitive {
					headers[key] = "[REDACTED]"
				} else if len(values) > 0 {
					headers[key] = values[0]
				}
			}
			fields = append(fields, logger.Any("headers", headers))
		}

		// Add errors if any
		if len(c.Errors) > 0 {
			errors := make([]string, len(c.Errors))
			for i, e := range c.Errors {
				errors[i] = e.Error()
			}
			fields = append(fields, logger.Strings("errors", errors))
		}

		// Log based on status code
		msg := "HTTP Request"
		switch {
		case statusCode >= 500:
			log.Error(msg, fields...)
		case statusCode >= 400:
			log.Warn(msg, fields...)
		default:
			log.Info(msg, fields...)
		}
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return time.Now().Format("20060102150405.000000000")
}

// GetRequestID retrieves the request ID from the gin context
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get("request_id"); exists {
		if reqID, ok := id.(string); ok {
			return reqID
		}
	}
	return ""
}

// GetTraceID retrieves the trace ID from the gin context
func GetTraceID(c *gin.Context) string {
	if id, exists := c.Get("trace_id"); exists {
		if traceID, ok := id.(string); ok {
			return traceID
		}
	}
	return ""
}

// GetSpanID retrieves the span ID from the gin context
func GetSpanID(c *gin.Context) string {
	if id, exists := c.Get("span_id"); exists {
		if spanID, ok := id.(string); ok {
			return spanID
		}
	}
	return ""
}
