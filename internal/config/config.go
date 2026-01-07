package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/bravo68web/stasis/internal/infrastructure/otel"
	"github.com/bravo68web/stasis/pkg/logger"
	"github.com/spf13/viper"
)

// Config represents the complete application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Storage  StorageConfig  `mapstructure:"storage"`
	SSH      SSHConfig      `mapstructure:"ssh"`
	OIDC     OIDCConfig     `mapstructure:"oidc"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	CI       CIConfig       `mapstructure:"ci"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug, release, test
}

// DatabaseConfig holds PostgreSQL database configuration
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// DSN returns the database connection string (libpq format)
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// URL returns the database connection URL (for tools like Atlas)
func (d *DatabaseConfig) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode,
	)
}

// StorageConfig holds storage backend configuration
type StorageConfig struct {
	Type        string `mapstructure:"type"` // filesystem, s3
	BasePath    string `mapstructure:"base_path"`
	S3Bucket    string `mapstructure:"s3_bucket"`
	S3Region    string `mapstructure:"s3_region"`
	S3AccessKey string `mapstructure:"s3_access_key"`
	S3SecretKey string `mapstructure:"s3_secret_key"`
	S3Endpoint  string `mapstructure:"s3_endpoint"` // For S3-compatible services
}

// IsS3 returns true if the storage type is S3
func (s *StorageConfig) IsS3() bool {
	return strings.ToLower(s.Type) == "s3"
}

// IsFilesystem returns true if the storage type is filesystem
func (s *StorageConfig) IsFilesystem() bool {
	return strings.ToLower(s.Type) == "filesystem" || s.Type == ""
}

// SSHConfig holds SSH server configuration
type SSHConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	HostKeyPath string `mapstructure:"host_key_path"`
}

// Address returns the SSH server address
func (s *SSHConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// OIDCConfig holds OpenID Connect configuration
type OIDCConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	IssuerURL    string   `mapstructure:"issuer_url"`    // OIDC provider's issuer URL (e.g., https://accounts.google.com)
	ClientID     string   `mapstructure:"client_id"`     // OAuth2 client ID
	ClientSecret string   `mapstructure:"client_secret"` // OAuth2 client secret
	RedirectURL  string   `mapstructure:"redirect_url"`  // Callback URL (e.g., http://localhost/api/v1/auth/oidc/callback)
	FrontendURL  string   `mapstructure:"frontend_url"`  // Frontend URL for redirecting after OIDC callback (e.g., http://localhost:3000)
	Scopes       []string `mapstructure:"scopes"`        // OIDC scopes (default: openid, profile, email)
	JWTSecret    string   `mapstructure:"jwt_secret"`    // Secret for signing session JWTs
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	// Level is the minimum log level (debug, info, warn, error)
	Level string `mapstructure:"level"`

	// Output defines where logs should be written (console, file, otel)
	Output string `mapstructure:"output"`

	// Format defines the log format (json, console) - only applicable for console/file output
	Format string `mapstructure:"format"`

	// FilePath is the path to the log file (required when Output is "file")
	FilePath string `mapstructure:"file_path"`

	// FileMaxSizeMB is the maximum size of the log file in megabytes before rotation
	FileMaxSizeMB int `mapstructure:"file_max_size_mb"`

	// FileMaxBackups is the maximum number of old log files to retain
	FileMaxBackups int `mapstructure:"file_max_backups"`

	// FileMaxAgeDays is the maximum number of days to retain old log files
	FileMaxAgeDays int `mapstructure:"file_max_age_days"`

	// FileCompress determines if rotated log files should be compressed
	FileCompress bool `mapstructure:"file_compress"`

	// Development enables development mode (more verbose, stacktraces, etc.)
	Development bool `mapstructure:"development"`

	// AddCaller adds caller information to log entries
	AddCaller bool `mapstructure:"add_caller"`

	// OTEL holds OpenTelemetry logging configuration
	OTEL OTELLoggingConfig `mapstructure:"otel"`
}

// OTELLoggingConfig holds OpenTelemetry logging configuration
type OTELLoggingConfig struct {
	// Enabled determines if OTEL logging is enabled
	Enabled bool `mapstructure:"enabled"`

	// Endpoint is the OTEL collector endpoint (e.g., "localhost:4317")
	Endpoint string `mapstructure:"endpoint"`

	// ServiceName is the name of the service for OTEL
	ServiceName string `mapstructure:"service_name"`

	// ServiceVersion is the version of the service
	ServiceVersion string `mapstructure:"service_version"`

	// Environment is the deployment environment (e.g., "production", "staging")
	Environment string `mapstructure:"environment"`

	// Insecure disables TLS for the OTEL connection
	Insecure bool `mapstructure:"insecure"`

	// Headers are additional headers to send with OTEL requests
	Headers map[string]string `mapstructure:"headers"`
}

// ToLoggerConfig converts LoggingConfig to a logger.Config
func (c *LoggingConfig) ToLoggerConfig() *logger.Config {
	var output logger.OutputType
	switch strings.ToLower(c.Output) {
	case "file":
		output = logger.OutputFile
	case "otel":
		output = logger.OutputOTEL
	default:
		output = logger.OutputConsole
	}

	return &logger.Config{
		Level:          c.Level,
		Output:         output,
		Format:         c.Format,
		FilePath:       c.FilePath,
		FileMaxSizeMB:  c.FileMaxSizeMB,
		FileMaxBackups: c.FileMaxBackups,
		FileMaxAgeDays: c.FileMaxAgeDays,
		FileCompress:   c.FileCompress,
		Development:    c.Development,
		AddCaller:      c.AddCaller,
		CallerSkip:     1,
	}
}

// ToOTELConfig converts OTELLoggingConfig to an otel.Config
func (c *OTELLoggingConfig) ToOTELConfig() *otel.Config {
	return &otel.Config{
		Enabled:        c.Enabled,
		Endpoint:       c.Endpoint,
		ServiceName:    c.ServiceName,
		ServiceVersion: c.ServiceVersion,
		Environment:    c.Environment,
		Insecure:       c.Insecure,
		Headers:        c.Headers,
	}
}

// Load reads configuration from file and environment variables
// It supports loading from:
// 1. Explicit file path (if provided and exists on filesystem)
// 2. Common filesystem locations
// 3. Environment variables (always applied as overrides)
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Set config type
	v.SetConfigType("yaml")

	// Read from environment variables
	v.SetEnvPrefix("STASIS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Try to load config file
	configLoaded := false

	// 1. Try explicit config path on filesystem first
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			v.SetConfigFile(configPath)
			if err := v.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			configLoaded = true
		}
	}

	// 2. Try common filesystem locations if still not loaded
	if !configLoaded {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/stasis")

		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			// Config file not found; rely on defaults and env vars
		}
	}

	// Override with environment variables for sensitive data
	overrideFromEnv(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "release")

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "stasis")
	v.SetDefault("database.password", "password")
	v.SetDefault("database.dbname", "stasis")
	v.SetDefault("database.sslmode", "disable")

	// Storage defaults
	v.SetDefault("storage.type", "filesystem")
	v.SetDefault("storage.base_path", "./data/repos")

	// SSH defaults
	v.SetDefault("ssh.enabled", true)
	v.SetDefault("ssh.host", "0.0.0.0")
	v.SetDefault("ssh.port", 2222)
	v.SetDefault("ssh.host_key_path", "./ssh_host_key")

	// OIDC defaults
	v.SetDefault("oidc.enabled", false)
	v.SetDefault("oidc.issuer_url", "")
	v.SetDefault("oidc.client_id", "")
	v.SetDefault("oidc.client_secret", "")
	v.SetDefault("oidc.redirect_url", "")
	v.SetDefault("oidc.frontend_url", "http://localhost:3000")
	v.SetDefault("oidc.scopes", []string{"openid", "profile", "email"})
	v.SetDefault("oidc.jwt_secret", "change-this-secret-in-production")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.output", "console")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.file_path", "./logs/app.log")
	v.SetDefault("logging.file_max_size_mb", 100)
	v.SetDefault("logging.file_max_backups", 3)
	v.SetDefault("logging.file_max_age_days", 28)
	v.SetDefault("logging.file_compress", true)
	v.SetDefault("logging.development", false)
	v.SetDefault("logging.add_caller", true)

	// OTEL logging defaults
	v.SetDefault("logging.otel.enabled", false)
	v.SetDefault("logging.otel.endpoint", "localhost:4317")
	v.SetDefault("logging.otel.service_name", "stasis")
	v.SetDefault("logging.otel.service_version", "1.0.0")
	v.SetDefault("logging.otel.environment", "development")
	v.SetDefault("logging.otel.insecure", true)

	// CI defaults
	v.SetDefault("ci.enabled", false)
	v.SetDefault("ci.server_url", "http://localhost:8081")
	v.SetDefault("ci.git_server_url", "")
	v.SetDefault("ci.api_key", "")
	v.SetDefault("ci.config_path", ".stasis-ci.yaml")
	v.SetDefault("ci.timeout", 30)
	v.SetDefault("ci.webhook_secret", "")
	v.SetDefault("ci.max_concurrent_jobs", 5)
	v.SetDefault("ci.retention_days", 30)
}

// overrideFromEnv handles special environment variable overrides
func overrideFromEnv(v *viper.Viper) {
	// Database password from env
	if dbPass := os.Getenv("STASIS_DB_PASSWORD"); dbPass != "" {
		v.Set("database.password", dbPass)
	}

	// S3 credentials from env (more secure than config file)
	if s3Key := os.Getenv("AWS_ACCESS_KEY_ID"); s3Key != "" {
		v.Set("storage.s3_access_key", s3Key)
	}
	if s3Secret := os.Getenv("AWS_SECRET_ACCESS_KEY"); s3Secret != "" {
		v.Set("storage.s3_secret_key", s3Secret)
	}

	// OIDC credentials from env (more secure than config file)
	if oidcClientID := os.Getenv("STASIS_OIDC_CLIENT_ID"); oidcClientID != "" {
		v.Set("oidc.client_id", oidcClientID)
	}
	if oidcClientSecret := os.Getenv("STASIS_OIDC_CLIENT_SECRET"); oidcClientSecret != "" {
		v.Set("oidc.client_secret", oidcClientSecret)
	}
	if oidcJWTSecret := os.Getenv("STASIS_OIDC_JWT_SECRET"); oidcJWTSecret != "" {
		v.Set("oidc.jwt_secret", oidcJWTSecret)
	}
	if oidcFrontendURL := os.Getenv("STASIS_OIDC_FRONTEND_URL"); oidcFrontendURL != "" {
		v.Set("oidc.frontend_url", oidcFrontendURL)
	}

	// CI credentials from env
	if ciAPIKey := os.Getenv("STASIS_CI_API_KEY"); ciAPIKey != "" {
		v.Set("ci.api_key", ciAPIKey)
	}
	if ciWebhookSecret := os.Getenv("STASIS_CI_WEBHOOK_SECRET"); ciWebhookSecret != "" {
		v.Set("ci.webhook_secret", ciWebhookSecret)
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Validate database config
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	// Validate storage config
	if c.Storage.IsS3() {
		if c.Storage.S3Bucket == "" {
			return fmt.Errorf("S3 bucket is required when using S3 storage")
		}
		if c.Storage.S3Region == "" {
			return fmt.Errorf("S3 region is required when using S3 storage")
		}
	} else if c.Storage.IsFilesystem() {
		if c.Storage.BasePath == "" {
			return fmt.Errorf("storage base path is required for filesystem storage")
		}
	} else {
		return fmt.Errorf("invalid storage type: %s", c.Storage.Type)
	}

	// Validate SSH config if enabled
	if c.SSH.Enabled {
		if c.SSH.Port <= 0 || c.SSH.Port > 65535 {
			return fmt.Errorf("invalid SSH port: %d", c.SSH.Port)
		}
	}

	// Validate OIDC config if enabled
	if c.OIDC.Enabled {
		if c.OIDC.IssuerURL == "" {
			return fmt.Errorf("OIDC issuer URL is required when OIDC is enabled")
		}
		// if c.OIDC.ClientSecret == "" {
		// 	return fmt.Errorf("OIDC client secret is required when OIDC is enabled")
		// }
		if c.OIDC.RedirectURL == "" {
			return fmt.Errorf("OIDC redirect URL is required when OIDC is enabled")
		}
		if c.OIDC.JWTSecret == "" {
			return fmt.Errorf("OIDC JWT secret is required when OIDC is enabled")
		}
	}

	return nil
}

// ServerAddress returns the HTTP server address
func (c *Config) ServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Mode == "debug" || c.Server.Mode == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Mode == "release" || c.Server.Mode == "production"
}
