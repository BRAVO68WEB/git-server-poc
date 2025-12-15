package config

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// EmbeddedFS can be set to use embedded configuration files
// This should be set from the configs package if embedding is desired
var EmbeddedFS embed.FS

// Config represents the complete application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Storage  StorageConfig  `mapstructure:"storage"`
	SSH      SSHConfig      `mapstructure:"ssh"`
	OPA      OPAConfig      `mapstructure:"opa"`
	Logging  LoggingConfig  `mapstructure:"logging"`
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

// DSN returns the database connection string
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
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

// OPAConfig holds Open Policy Agent configuration
type OPAConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	PolicyPath string `mapstructure:"policy_path"` // Path to .rego policy file
	Query      string `mapstructure:"query"`       // OPA query (default: data.gitserver.authz.allow)
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"` // debug, info, warn, error
	OutputPath string `mapstructure:"output_path"`
	Format     string `mapstructure:"format"` // json, console
}

// Load reads configuration from file and environment variables
// It supports loading from:
// 1. Explicit file path (if provided and exists on filesystem)
// 2. Embedded filesystem (if EmbeddedFS is set)
// 3. Common filesystem locations
// 4. Environment variables (always applied as overrides)
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Set config type
	v.SetConfigType("yaml")

	// Read from environment variables
	v.SetEnvPrefix("GITSERVER")
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

	// 2. Try embedded filesystem if config not loaded and EmbeddedFS is set
	if !configLoaded {
		embeddedConfig, err := tryLoadEmbeddedConfig(configPath)
		if err == nil && embeddedConfig != nil {
			if err := v.ReadConfig(bytes.NewReader(embeddedConfig)); err != nil {
				return nil, fmt.Errorf("failed to read embedded config: %w", err)
			}
			configLoaded = true
		}
	}

	// 3. Try common filesystem locations if still not loaded
	if !configLoaded {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/git-server")

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

// LoadWithEmbedded loads configuration with an embedded filesystem
// This is a convenience function for use with embedded configs
func LoadWithEmbedded(configPath string, embeddedFS embed.FS) (*Config, error) {
	EmbeddedFS = embeddedFS
	return Load(configPath)
}

// tryLoadEmbeddedConfig attempts to load config from the embedded filesystem
func tryLoadEmbeddedConfig(configPath string) ([]byte, error) {
	// Check if EmbeddedFS has any files
	entries, err := fs.ReadDir(EmbeddedFS, ".")
	if err != nil || len(entries) == 0 {
		return nil, fmt.Errorf("no embedded config available")
	}

	// Try the specific config path first (strip directory prefix if present)
	if configPath != "" {
		// Try various path formats
		pathsToTry := []string{
			configPath,
			strings.TrimPrefix(configPath, "configs/"),
			strings.TrimPrefix(configPath, "./configs/"),
			strings.TrimPrefix(configPath, "./"),
		}

		for _, path := range pathsToTry {
			if data, err := fs.ReadFile(EmbeddedFS, path); err == nil {
				return data, nil
			}
		}
	}

	// Try default config names
	defaultNames := []string{"config.yaml", "config.yml"}
	for _, name := range defaultNames {
		if data, err := fs.ReadFile(EmbeddedFS, name); err == nil {
			return data, nil
		}
	}

	return nil, fmt.Errorf("config file not found in embedded filesystem")
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
	v.SetDefault("database.user", "gitserver")
	v.SetDefault("database.password", "password")
	v.SetDefault("database.dbname", "gitserver")
	v.SetDefault("database.sslmode", "disable")

	// Storage defaults
	v.SetDefault("storage.type", "filesystem")
	v.SetDefault("storage.base_path", "./data/repos")

	// SSH defaults
	v.SetDefault("ssh.enabled", true)
	v.SetDefault("ssh.host", "0.0.0.0")
	v.SetDefault("ssh.port", 2222)
	v.SetDefault("ssh.host_key_path", "./ssh_host_key")

	// OPA defaults (using embedded Go SDK)
	v.SetDefault("opa.enabled", false)
	v.SetDefault("opa.policy_path", "./policies/rbac.rego")
	v.SetDefault("opa.query", "data.gitserver.authz.allow")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.output_path", "stdout")
	v.SetDefault("logging.format", "json")
}

// overrideFromEnv handles special environment variable overrides
func overrideFromEnv(v *viper.Viper) {
	// Database password from env
	if dbPass := os.Getenv("GITSERVER_DB_PASSWORD"); dbPass != "" {
		v.Set("database.password", dbPass)
	}

	// S3 credentials from env (more secure than config file)
	if s3Key := os.Getenv("AWS_ACCESS_KEY_ID"); s3Key != "" {
		v.Set("storage.s3_access_key", s3Key)
	}
	if s3Secret := os.Getenv("AWS_SECRET_ACCESS_KEY"); s3Secret != "" {
		v.Set("storage.s3_secret_key", s3Secret)
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

	// Validate OPA config if enabled
	if c.OPA.Enabled {
		if c.OPA.PolicyPath == "" {
			return fmt.Errorf("OPA policy path is required when OPA is enabled")
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
