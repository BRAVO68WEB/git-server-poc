package config

import (
	"fmt"
	"strings"
	"time"
)

// CIConfig holds CI/CD runner integration configuration
type CIConfig struct {
	// Enabled determines if CI/CD integration is active
	Enabled bool `mapstructure:"enabled"`

	// ServerURL is the base URL of the CI runner server
	// e.g., http://localhost:8081 or http://ci-runner:8081
	ServerURL string `mapstructure:"server_url"`

	// GitServerURL is the URL the CI runner should use to clone repositories
	// This should be accessible from the CI runner's network
	// If empty, defaults to the main server's hosted URL
	GitServerURL string `mapstructure:"git_server_url"`

	// APIKey is the API key for authenticating with the CI server
	// Should be set via environment variable STASIS_CI_API_KEY in production
	APIKey string `mapstructure:"api_key"`

	// ConfigPath is the path to the CI config file in repositories
	// Default: ".stasis-ci.yaml"
	ConfigPath string `mapstructure:"config_path"`

	// TimeoutSeconds is the timeout for CI server requests in seconds
	TimeoutSeconds int `mapstructure:"timeout"`

	// WebhookSecret is the secret for validating webhooks from CI runner
	WebhookSecret string `mapstructure:"webhook_secret"`

	// MaxConcurrentJobs is the maximum number of concurrent jobs per repository
	MaxConcurrentJobs int `mapstructure:"max_concurrent_jobs"`

	// RetentionDays is how long to keep job history
	RetentionDays int `mapstructure:"retention_days"`
}

// DefaultCIConfig returns default CI configuration
func DefaultCIConfig() CIConfig {
	return CIConfig{
		Enabled:           false,
		ServerURL:         "http://localhost:8081",
		GitServerURL:      "",
		APIKey:            "",
		ConfigPath:        ".stasis-ci.yaml",
		TimeoutSeconds:    30,
		WebhookSecret:     "",
		MaxConcurrentJobs: 5,
		RetentionDays:     30,
	}
}

// IsConfigured returns true if CI is enabled and properly configured
func (c *CIConfig) IsConfigured() bool {
	return c.Enabled && c.ServerURL != ""
}

// Timeout returns the timeout as a time.Duration
func (c *CIConfig) Timeout() time.Duration {
	if c.TimeoutSeconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// GetGitServerURL returns the Git server URL for CI runner to use
// Falls back to empty string if not configured (caller should use hosted_url)
func (c *CIConfig) GetGitServerURL() string {
	if c.GitServerURL != "" {
		return c.GitServerURL
	}
	return ""
}

// GetConfigPath returns the CI config path with default fallback
func (c *CIConfig) GetConfigPath() string {
	if c.ConfigPath != "" {
		return c.ConfigPath
	}
	return ".stasis-ci.yaml"
}

func (c *CIConfig) GetAPIToken() string {
	if c.APIKey != "" {
		return c.APIKey
	}
	return ""
}

func (c *CIConfig) GetGitServerURLWithAPIToken() string {
	baseURL := c.GetGitServerURL()
	parts := strings.Split(baseURL, "://")
	if len(parts) != 2 {
		return baseURL
	}
	return fmt.Sprintf("%s://oauth:%s@%s", parts[0], c.GetAPIToken(), parts[1])
}
