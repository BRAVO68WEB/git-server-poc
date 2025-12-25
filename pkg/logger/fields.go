package logger

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Field type alias for convenience
type Field = zap.Field

// Common field constructors - re-exported from zap for convenience

// String constructs a field with the given key and value
func String(key string, val string) Field {
	return zap.String(key, val)
}

// Strings constructs a field with the given key and slice of strings
func Strings(key string, val []string) Field {
	return zap.Strings(key, val)
}

// Int constructs a field with the given key and value
func Int(key string, val int) Field {
	return zap.Int(key, val)
}

// Int64 constructs a field with the given key and value
func Int64(key string, val int64) Field {
	return zap.Int64(key, val)
}

// Int32 constructs a field with the given key and value
func Int32(key string, val int32) Field {
	return zap.Int32(key, val)
}

// Uint constructs a field with the given key and value
func Uint(key string, val uint) Field {
	return zap.Uint(key, val)
}

// Uint64 constructs a field with the given key and value
func Uint64(key string, val uint64) Field {
	return zap.Uint64(key, val)
}

// Float64 constructs a field with the given key and value
func Float64(key string, val float64) Field {
	return zap.Float64(key, val)
}

// Float32 constructs a field with the given key and value
func Float32(key string, val float32) Field {
	return zap.Float32(key, val)
}

// Bool constructs a field with the given key and value
func Bool(key string, val bool) Field {
	return zap.Bool(key, val)
}

// Time constructs a field with the given key and value
func Time(key string, val time.Time) Field {
	return zap.Time(key, val)
}

// Duration constructs a field with the given key and value
func Duration(key string, val time.Duration) Field {
	return zap.Duration(key, val)
}

// Error constructs a field that lazily stores err.Error() under the key "error"
func Error(err error) Field {
	return zap.Error(err)
}

// NamedError constructs a field that lazily stores err.Error() under the provided key
func NamedError(key string, err error) Field {
	return zap.NamedError(key, err)
}

// Any takes a key and an arbitrary value and chooses the best way to represent them
func Any(key string, val interface{}) Field {
	return zap.Any(key, val)
}

// Binary constructs a field that carries an opaque binary blob
func Binary(key string, val []byte) Field {
	return zap.Binary(key, val)
}

// ByteString constructs a field that carries UTF-8 encoded text as a []byte
func ByteString(key string, val []byte) Field {
	return zap.ByteString(key, val)
}

// Stringer constructs a field with the given key and the output of the value's String method
func Stringer(key string, val fmt.Stringer) Field {
	return zap.Stringer(key, val)
}

// Reflect constructs a field by running reflection over the provided value
func Reflect(key string, val interface{}) Field {
	return zap.Reflect(key, val)
}

// Namespace creates a named, isolated scope within the logger's context
func Namespace(key string) Field {
	return zap.Namespace(key)
}

// Stack constructs a field that stores a stacktrace of the current goroutine
func Stack(key string) Field {
	return zap.Stack(key)
}

// StackSkip constructs a field similarly to Stack, but also skips the given number of frames
func StackSkip(key string, skip int) Field {
	return zap.StackSkip(key, skip)
}

// Object constructs a field with the given key and ObjectMarshaler
func Object(key string, val zapcore.ObjectMarshaler) Field {
	return zap.Object(key, val)
}

// Array constructs a field with the given key and ArrayMarshaler
func Array(key string, val zapcore.ArrayMarshaler) Field {
	return zap.Array(key, val)
}

// Skip constructs a no-op field, which is often useful when handling invalid inputs
func Skip() Field {
	return zap.Skip()
}

// HTTP Request related fields

// RequestID constructs a field for request ID
func RequestID(id string) Field {
	return String("request_id", id)
}

// TraceID constructs a field for trace ID (OTEL)
func TraceID(id string) Field {
	return String("trace_id", id)
}

// SpanID constructs a field for span ID (OTEL)
func SpanID(id string) Field {
	return String("span_id", id)
}

// UserID constructs a field for user ID
func UserID(id string) Field {
	return String("user_id", id)
}

// Method constructs a field for HTTP method
func Method(method string) Field {
	return String("method", method)
}

// Path constructs a field for URL path
func Path(path string) Field {
	return String("path", path)
}

// StatusCode constructs a field for HTTP status code
func StatusCode(code int) Field {
	return Int("status_code", code)
}

// Latency constructs a field for request latency
func Latency(d time.Duration) Field {
	return Duration("latency", d)
}

// ClientIP constructs a field for client IP address
func ClientIP(ip string) Field {
	return String("client_ip", ip)
}

// UserAgent constructs a field for user agent
func UserAgent(ua string) Field {
	return String("user_agent", ua)
}

// Component constructs a field for component name
func Component(name string) Field {
	return String("component", name)
}

// Operation constructs a field for operation name
func Operation(name string) Field {
	return String("operation", name)
}

// Service constructs a field for service name
func Service(name string) Field {
	return String("service", name)
}

// Version constructs a field for version
func Version(version string) Field {
	return String("version", version)
}

// Environment constructs a field for environment
func Environment(env string) Field {
	return String("environment", env)
}

// Repository constructs a field for repository name (git-specific)
func Repository(name string) Field {
	return String("repository", name)
}

// Branch constructs a field for branch name (git-specific)
func Branch(name string) Field {
	return String("branch", name)
}

// Commit constructs a field for commit hash (git-specific)
func Commit(hash string) Field {
	return String("commit", hash)
}

// Query constructs a field for URL query string
func Query(q string) Field {
	return String("query", q)
}

// BodySize constructs a field for response body size
func BodySize(size int) Field {
	return Int("body_size", size)
}

// Protocol constructs a field for protocol (HTTP/1.1, HTTP/2, etc.)
func Protocol(proto string) Field {
	return String("protocol", proto)
}

// Referer constructs a field for HTTP referer header
func Referer(ref string) Field {
	return String("referer", ref)
}
