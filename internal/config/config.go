package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the complete application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Storage  StorageConfig  `mapstructure:"storage"`
	SSH      SSHConfig      `mapstructure:"ssh"`
	OIDC     OIDCConfig     `mapstructure:"oidc"`
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

// OIDCConfig holds OpenID Connect configuration
type OIDCConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	IssuerURL    string   `mapstructure:"issuer_url"`    // OIDC provider's issuer URL (e.g., https://accounts.google.com)
	ClientID     string   `mapstructure:"client_id"`     // OAuth2 client ID
	ClientSecret string   `mapstructure:"client_secret"` // OAuth2 client secret
	RedirectURL  string   `mapstructure:"redirect_url"`  // Callback URL (e.g., http://localhost:8080/api/v1/auth/oidc/callback)
	FrontendURL  string   `mapstructure:"frontend_url"`  // Frontend URL for redirecting after OIDC callback (e.g., http://localhost:3000)
	Scopes       []string `mapstructure:"scopes"`        // OIDC scopes (default: openid, profile, email)
	JWTSecret    string   `mapstructure:"jwt_secret"`    // Secret for signing session JWTs
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
// 2. Common filesystem locations
// 3. Environment variables (always applied as overrides)
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

	// 2. Try common filesystem locations if still not loaded
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

	// OIDC defaults
	v.SetDefault("oidc.enabled", false)
	v.SetDefault("oidc.issuer_url", "")
	v.SetDefault("oidc.client_id", "")
	v.SetDefault("oidc.client_secret", "")
	v.SetDefault("oidc.redirect_url", "")
	v.SetDefault("oidc.frontend_url", "http://localhost:3000")
	v.SetDefault("oidc.scopes", []string{"openid", "profile", "email"})
	v.SetDefault("oidc.jwt_secret", "change-this-secret-in-production")

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

	// OIDC credentials from env (more secure than config file)
	if oidcClientID := os.Getenv("GITSERVER_OIDC_CLIENT_ID"); oidcClientID != "" {
		v.Set("oidc.client_id", oidcClientID)
	}
	if oidcClientSecret := os.Getenv("GITSERVER_OIDC_CLIENT_SECRET"); oidcClientSecret != "" {
		v.Set("oidc.client_secret", oidcClientSecret)
	}
	if oidcJWTSecret := os.Getenv("GITSERVER_OIDC_JWT_SECRET"); oidcJWTSecret != "" {
		v.Set("oidc.jwt_secret", oidcJWTSecret)
	}
	if oidcFrontendURL := os.Getenv("GITSERVER_OIDC_FRONTEND_URL"); oidcFrontendURL != "" {
		v.Set("oidc.frontend_url", oidcFrontendURL)
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
