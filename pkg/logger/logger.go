package logger

import (
	"context"
	"io"
	"os"
	"sync"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// OutputType defines the type of output for the logger
type OutputType string

const (
	// OutputConsole outputs logs to the console (stdout/stderr)
	OutputConsole OutputType = "console"
	// OutputFile outputs logs to a file
	OutputFile OutputType = "file"
	// OutputOTEL outputs logs to OpenTelemetry collector
	OutputOTEL OutputType = "otel"
)

// Config holds the logger configuration
type Config struct {
	// Level is the minimum log level (debug, info, warn, error)
	Level string

	// Output defines where logs should be written (console, file, otel)
	Output OutputType

	// Format defines the log format (json, console) - only applicable for console/file output
	Format string

	// FilePath is the path to the log file (required when Output is "file")
	FilePath string

	// FileMaxSizeMB is the maximum size of the log file in megabytes before rotation
	FileMaxSizeMB int

	// FileMaxBackups is the maximum number of old log files to retain
	FileMaxBackups int

	// FileMaxAgeDays is the maximum number of days to retain old log files
	FileMaxAgeDays int

	// FileCompress determines if rotated log files should be compressed
	FileCompress bool

	// Development enables development mode (more verbose, stacktraces, etc.)
	Development bool

	// AddCaller adds caller information to log entries
	AddCaller bool

	// CallerSkip is the number of stack frames to skip when recording caller info
	CallerSkip int
}

// DefaultConfig returns a default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:          "info",
		Output:         OutputConsole,
		Format:         "json",
		FilePath:       "./logs/app.log",
		FileMaxSizeMB:  100,
		FileMaxBackups: 3,
		FileMaxAgeDays: 28,
		FileCompress:   true,
		Development:    false,
		AddCaller:      true,
		CallerSkip:     1,
	}
}

// Logger wraps zap.Logger with additional functionality
type Logger struct {
	*zap.Logger
	sugar   *zap.SugaredLogger
	config  *Config
	core    zapcore.Core
	closers []io.Closer
	mu      sync.RWMutex
}

var (
	globalLogger *Logger
	globalMu     sync.RWMutex
)

// New creates a new Logger instance based on the provided configuration
func New(cfg *Config) (*Logger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Parse log level
	level, err := parseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// Create encoder config
	encoderConfig := createEncoderConfig(cfg.Development)

	// Create the core based on output type
	var core zapcore.Core

	switch cfg.Output {
	case OutputFile:
		core, err = createFileCore(cfg, level, encoderConfig)
		if err != nil {
			return nil, err
		}
	default: // OutputConsole
		core = createConsoleCore(cfg, level, encoderConfig)
	}

	// Build zap options
	opts := buildZapOptions(cfg)

	// Create the zap logger
	zapLogger := zap.New(core, opts...)

	logger := &Logger{
		Logger:  zapLogger,
		sugar:   zapLogger.Sugar(),
		config:  cfg,
		core:    core,
		closers: make([]io.Closer, 0),
	}

	return logger, nil
}

// NewWithCore creates a new Logger with a custom zapcore.Core
// This is used for OTEL integration
func NewWithCore(cfg *Config, core zapcore.Core, closers ...io.Closer) *Logger {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	opts := buildZapOptions(cfg)
	zapLogger := zap.New(core, opts...)

	return &Logger{
		Logger:  zapLogger,
		sugar:   zapLogger.Sugar(),
		config:  cfg,
		core:    core,
		closers: closers,
	}
}

// Init initializes the global logger with the provided configuration
func Init(cfg *Config) error {
	logger, err := New(cfg)
	if err != nil {
		return err
	}

	SetGlobal(logger)
	return nil
}

// SetGlobal sets the global logger instance
func SetGlobal(logger *Logger) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = logger
}

// Get returns the global logger instance
func Get() *Logger {
	globalMu.RLock()
	if globalLogger != nil {
		defer globalMu.RUnlock()
		return globalLogger
	}
	globalMu.RUnlock()

	// Initialize with default config if not set
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalLogger == nil {
		cfg := DefaultConfig()
		logger, _ := New(cfg)
		globalLogger = logger
	}

	return globalLogger
}

// Sugar returns the sugared logger
func (l *Logger) Sugar() *zap.SugaredLogger {
	return l.sugar
}

// Core returns the underlying zapcore.Core
func (l *Logger) Core() zapcore.Core {
	return l.core
}

// WithContext returns a logger with trace information from the context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}

	// Extract trace and span IDs from the context
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return l
	}

	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()

	newLogger := l.With(
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
	)

	return &Logger{
		Logger:  newLogger,
		sugar:   newLogger.Sugar(),
		config:  l.config,
		core:    l.core,
		closers: l.closers,
	}
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields ...zap.Field) *Logger {
	newLogger := l.With(fields...)
	return &Logger{
		Logger:  newLogger,
		sugar:   newLogger.Sugar(),
		config:  l.config,
		core:    l.core,
		closers: l.closers,
	}
}

// WithError returns a logger with an error field
func (l *Logger) WithError(err error) *Logger {
	return l.WithFields(zap.Error(err))
}

// Close closes the logger and flushes any buffered logs
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Sync the logger to flush any buffered entries
	_ = l.Logger.Sync()

	// Close all closers
	var lastErr error
	for _, closer := range l.closers {
		if err := closer.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// parseLevel converts a string level to zapcore.Level
func parseLevel(level string) (zapcore.Level, error) {
	var l zapcore.Level
	err := l.UnmarshalText([]byte(level))
	return l, err
}

// createEncoderConfig creates the encoder configuration
func createEncoderConfig(development bool) zapcore.EncoderConfig {
	if development {
		config := zap.NewDevelopmentEncoderConfig()
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		return config
	}

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.TimeKey = "timestamp"
	config.MessageKey = "message"
	config.LevelKey = "level"
	config.CallerKey = "caller"
	config.StacktraceKey = "stacktrace"
	return config
}

// createConsoleCore creates a core for console output
func createConsoleCore(cfg *Config, level zapcore.Level, encoderConfig zapcore.EncoderConfig) zapcore.Core {
	var encoder zapcore.Encoder
	if cfg.Format == "console" || cfg.Development {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	return zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)
}

// createFileCore creates a core for file output
func createFileCore(cfg *Config, level zapcore.Level, encoderConfig zapcore.EncoderConfig) (zapcore.Core, error) {
	// Ensure the log directory exists
	if err := ensureLogDir(cfg.FilePath); err != nil {
		return nil, err
	}

	// Create file writer with rotation support
	writer := &fileWriter{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.FileMaxSizeMB,
		MaxBackups: cfg.FileMaxBackups,
		MaxAge:     cfg.FileMaxAgeDays,
		Compress:   cfg.FileCompress,
	}

	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	return zapcore.NewCore(
		encoder,
		zapcore.AddSync(writer),
		level,
	), nil
}

// buildZapOptions builds the zap.Option slice based on configuration
func buildZapOptions(cfg *Config) []zap.Option {
	var opts []zap.Option

	if cfg.AddCaller {
		opts = append(opts, zap.AddCaller())
		if cfg.CallerSkip > 0 {
			opts = append(opts, zap.AddCallerSkip(cfg.CallerSkip))
		}
	}

	if cfg.Development {
		opts = append(opts, zap.Development())
		opts = append(opts, zap.AddStacktrace(zapcore.WarnLevel))
	} else {
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	return opts
}

// Global helper functions

// Debug logs a debug message using the global logger
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info logs an info message using the global logger
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn logs a warning message using the global logger
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Errorf logs an error message using the global logger
func Errorf(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal logs a fatal message and exits using the global logger
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// Panic logs a panic message and panics using the global logger
func Panic(msg string, fields ...zap.Field) {
	Get().Panic(msg, fields...)
}

// With returns a logger with additional fields using the global logger
func With(fields ...zap.Field) *Logger {
	return Get().WithFields(fields...)
}

// WithContext returns a logger with trace context using the global logger
func WithContext(ctx context.Context) *Logger {
	return Get().WithContext(ctx)
}

// WithError returns a logger with an error field using the global logger
func WithErr(err error) *Logger {
	return Get().WithError(err)
}

// SyncGlobal flushes any buffered log entries from the global logger
func SyncGlobal() error {
	return Get().Sync()
}

// Close closes the global logger
func Close() error {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalLogger != nil {
		return globalLogger.Close()
	}
	return nil
}
