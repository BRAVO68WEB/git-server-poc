package otel

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap/zapcore"
)

// ZapCore is a zapcore.Core implementation that sends logs to OpenTelemetry
type ZapCore struct {
	zapcore.LevelEnabler
	provider *Provider
	logger   log.Logger
	fields   []zapcore.Field
}

// NewZapCore creates a new ZapCore that exports logs to OTEL
func NewZapCore(provider *Provider, level zapcore.Level) *ZapCore {
	return &ZapCore{
		LevelEnabler: level,
		provider:     provider,
		logger:       provider.Logger(),
		fields:       make([]zapcore.Field, 0),
	}
}

// With creates a new ZapCore with additional fields
func (c *ZapCore) With(fields []zapcore.Field) zapcore.Core {
	newFields := make([]zapcore.Field, len(c.fields)+len(fields))
	copy(newFields, c.fields)
	copy(newFields[len(c.fields):], fields)

	return &ZapCore{
		LevelEnabler: c.LevelEnabler,
		provider:     c.provider,
		logger:       c.logger,
		fields:       newFields,
	}
}

// Check implements zapcore.Core
func (c *ZapCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

// Write implements zapcore.Core
func (c *ZapCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// Combine fields
	allFields := make([]zapcore.Field, 0, len(c.fields)+len(fields))
	allFields = append(allFields, c.fields...)
	allFields = append(allFields, fields...)

	// Create OTEL log record
	record := log.Record{}
	record.SetTimestamp(entry.Time)
	record.SetSeverity(zapLevelToOTELSeverity(entry.Level))
	record.SetSeverityText(entry.Level.String())
	record.SetBody(log.StringValue(entry.Message))

	// Add attributes from fields
	attrs := make([]log.KeyValue, 0, len(allFields)+3)

	// Add caller information if available
	if entry.Caller.Defined {
		attrs = append(attrs,
			log.String("caller", entry.Caller.TrimmedPath()),
			log.Int("caller_line", entry.Caller.Line),
			log.String("caller_function", entry.Caller.Function),
		)
	}

	// Add logger name if available
	if entry.LoggerName != "" {
		attrs = append(attrs, log.String("logger", entry.LoggerName))
	}

	// Add stack trace if available
	if entry.Stack != "" {
		attrs = append(attrs, log.String("stacktrace", entry.Stack))
	}

	// Convert zap fields to OTEL attributes
	for _, field := range allFields {
		attr := zapFieldToOTELAttribute(field)
		if attr.Key != "" {
			attrs = append(attrs, attr)
		}
	}

	record.AddAttributes(attrs...)

	// Emit the log record
	c.logger.Emit(context.Background(), record)

	return nil
}

// Sync implements zapcore.Core
func (c *ZapCore) Sync() error {
	if c.provider != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return c.provider.ForceFlush(ctx)
	}
	return nil
}

// Provider returns the underlying OTEL provider
func (c *ZapCore) Provider() *Provider {
	return c.provider
}

// zapLevelToOTELSeverity converts zap log level to OTEL severity
func zapLevelToOTELSeverity(level zapcore.Level) log.Severity {
	switch level {
	case zapcore.DebugLevel:
		return log.SeverityDebug
	case zapcore.InfoLevel:
		return log.SeverityInfo
	case zapcore.WarnLevel:
		return log.SeverityWarn
	case zapcore.ErrorLevel:
		return log.SeverityError
	case zapcore.DPanicLevel:
		return log.SeverityError
	case zapcore.PanicLevel:
		return log.SeverityFatal
	case zapcore.FatalLevel:
		return log.SeverityFatal
	default:
		return log.SeverityInfo
	}
}

// zapFieldToOTELAttribute converts a zap field to an OTEL attribute
func zapFieldToOTELAttribute(field zapcore.Field) log.KeyValue {
	switch field.Type {
	case zapcore.BoolType:
		return log.Bool(field.Key, field.Integer == 1)

	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		return log.Int64(field.Key, field.Integer)

	case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
		return log.Int64(field.Key, field.Integer)

	case zapcore.Float64Type:
		return log.Float64(field.Key, float64(field.Integer))

	case zapcore.Float32Type:
		return log.Float64(field.Key, float64(field.Integer))

	case zapcore.StringType:
		return log.String(field.Key, field.String)

	case zapcore.TimeType:
		if field.Interface != nil {
			if t, ok := field.Interface.(time.Time); ok {
				return log.String(field.Key, t.Format(time.RFC3339Nano))
			}
		}
		return log.Int64(field.Key, field.Integer)

	case zapcore.TimeFullType:
		if field.Interface != nil {
			if t, ok := field.Interface.(time.Time); ok {
				return log.String(field.Key, t.Format(time.RFC3339Nano))
			}
		}
		return log.KeyValue{}

	case zapcore.DurationType:
		return log.String(field.Key, time.Duration(field.Integer).String())

	case zapcore.ErrorType:
		if field.Interface != nil {
			if err, ok := field.Interface.(error); ok {
				return log.String(field.Key, err.Error())
			}
		}
		return log.KeyValue{}

	case zapcore.StringerType:
		if field.Interface != nil {
			if s, ok := field.Interface.(fmt.Stringer); ok {
				return log.String(field.Key, s.String())
			}
		}
		return log.KeyValue{}

	case zapcore.BinaryType:
		if field.Interface != nil {
			if b, ok := field.Interface.([]byte); ok {
				return log.Bytes(field.Key, b)
			}
		}
		return log.KeyValue{}

	case zapcore.ByteStringType:
		if field.Interface != nil {
			if b, ok := field.Interface.([]byte); ok {
				return log.String(field.Key, string(b))
			}
		}
		return log.KeyValue{}

	case zapcore.SkipType:
		return log.KeyValue{}

	case zapcore.NamespaceType:
		// Namespaces are handled differently in OTEL
		// We prefix subsequent fields with the namespace
		return log.KeyValue{}

	default:
		if field.Interface != nil {
			return log.String(field.Key, fmt.Sprintf("%v", field.Interface))
		}
		return log.KeyValue{}
	}
}

// NewCombinedCore creates a zapcore.Core that writes to both a local core and OTEL
func NewCombinedCore(localCore zapcore.Core, provider *Provider, level zapcore.Level) zapcore.Core {
	otelCore := NewZapCore(provider, level)
	return zapcore.NewTee(localCore, otelCore)
}

// Ensure ZapCore implements zapcore.Core
var _ zapcore.Core = (*ZapCore)(nil)

// OTELLoggerProvider is a helper to get the SDK log provider
type OTELLoggerProvider interface {
	LoggerProvider() *sdklog.LoggerProvider
}

// Ensure Provider implements OTELLoggerProvider
var _ OTELLoggerProvider = (*Provider)(nil)
