package otel

import (
	"context"
	"fmt"
	"io"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds OpenTelemetry provider configuration
type Config struct {
	// Enabled determines if OTEL is enabled
	Enabled bool

	// Endpoint is the OTEL collector endpoint (e.g., "localhost:4317")
	Endpoint string

	// ServiceName is the name of the service for OTEL
	ServiceName string

	// ServiceVersion is the version of the service
	ServiceVersion string

	// Environment is the deployment environment (e.g., "production", "staging")
	Environment string

	// Insecure disables TLS for the OTEL connection
	Insecure bool

	// UseHTTP uses HTTP instead of gRPC for the OTEL exporter
	UseHTTP bool

	// Headers are additional headers to send with OTEL requests
	Headers map[string]string

	// BatchTimeout is the maximum time to wait before sending a batch
	BatchTimeout time.Duration
}

// DefaultConfig returns a default OTEL configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:        false,
		Endpoint:       "localhost:4317",
		ServiceName:    "stasis",
		ServiceVersion: "0.1.0",
		Environment:    "development",
		Insecure:       true,
		UseHTTP:        false,
		Headers:        make(map[string]string),
		BatchTimeout:   5 * time.Second,
	}
}

// Provider manages OpenTelemetry log provider
type Provider struct {
	config      *Config
	logProvider *sdklog.LoggerProvider
	logger      log.Logger
	resource    *resource.Resource
}

// NewProvider creates a new OTEL provider
func NewProvider(cfg *Config) (*Provider, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if !cfg.Enabled {
		return nil, fmt.Errorf("OTEL is not enabled")
	}

	ctx := context.Background()

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create the OTLP exporter
	exporter, err := createExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create batch processor options
	batchOpts := []sdklog.BatchProcessorOption{}
	if cfg.BatchTimeout > 0 {
		batchOpts = append(batchOpts, sdklog.WithExportTimeout(cfg.BatchTimeout))
	}

	// Create the log provider
	logProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter, batchOpts...)),
	)

	// Create the logger from provider
	logger := logProvider.Logger(cfg.ServiceName)

	return &Provider{
		config:      cfg,
		logProvider: logProvider,
		logger:      logger,
		resource:    res,
	}, nil
}

// createExporter creates an OTLP log exporter based on configuration
func createExporter(ctx context.Context, cfg *Config) (sdklog.Exporter, error) {
	if cfg.UseHTTP {
		return createHTTPExporter(ctx, cfg)
	}
	return createGRPCExporter(ctx, cfg)
}

// createGRPCExporter creates a gRPC-based OTLP exporter
func createGRPCExporter(ctx context.Context, cfg *Config) (sdklog.Exporter, error) {
	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		conn, err := grpc.NewClient(
			cfg.Endpoint,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
		}
		opts = []otlploggrpc.Option{
			otlploggrpc.WithGRPCConn(conn),
		}
	}

	return otlploggrpc.New(ctx, opts...)
}

// createHTTPExporter creates an HTTP-based OTLP exporter
func createHTTPExporter(ctx context.Context, cfg *Config) (sdklog.Exporter, error) {
	opts := []otlploghttp.Option{
		otlploghttp.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	}

	// Add custom headers if provided
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlploghttp.WithHeaders(cfg.Headers))
	}

	return otlploghttp.New(ctx, opts...)
}

// Logger returns the OTEL logger
func (p *Provider) Logger() log.Logger {
	return p.logger
}

// LoggerProvider returns the underlying log provider
func (p *Provider) LoggerProvider() *sdklog.LoggerProvider {
	return p.logProvider
}

// Resource returns the OTEL resource
func (p *Provider) Resource() *resource.Resource {
	return p.resource
}

// Config returns the provider configuration
func (p *Provider) Config() *Config {
	return p.config
}

// Shutdown gracefully shuts down the provider
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.logProvider != nil {
		return p.logProvider.Shutdown(ctx)
	}
	return nil
}

// ForceFlush forces a flush of all pending logs
func (p *Provider) ForceFlush(ctx context.Context) error {
	if p.logProvider != nil {
		return p.logProvider.ForceFlush(ctx)
	}
	return nil
}

// Close implements io.Closer for the provider
func (p *Provider) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return p.Shutdown(ctx)
}

// Ensure Provider implements io.Closer
var _ io.Closer = (*Provider)(nil)
