package config

import (
	"io"
	"os"
	"strings"
	"gopkg.in/yaml.v3"
)

type Config struct {
	PostgresDSN        string `yaml:"postgres_dsn"`
	S3Region           string `yaml:"s3_region"`
	S3Bucket           string `yaml:"s3_bucket"`
	S3Endpoint         string `yaml:"s3_endpoint"`
	AWSAccessKeyID     string `yaml:"aws_access_key_id"`
	AWSSecretAccessKey string `yaml:"aws_secret_access_key"`
	AWSSessionToken    string `yaml:"aws_session_token"`
	HTTPAddr           string `yaml:"http_addr"`
	SSHAddr            string `yaml:"ssh_addr"`
}

var cfgPathOverride string

func SetConfigPath(p string) {
	cfgPathOverride = strings.TrimSpace(p)
}

func LoadFromFile(path string) (Config, error) {
	var c Config
	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return c, err
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return c, err
	}
	return c, nil
}

func Load() Config {
	if cfgPathOverride != "" {
		if c, err := LoadFromFile(cfgPathOverride); err == nil {
			return c
		}
	}
	return Config{
		PostgresDSN: os.Getenv("GITHUT_POSTGRES_DSN"),
		S3Region:    os.Getenv("GITHUT_S3_REGION"),
		S3Bucket:    os.Getenv("GITHUT_S3_BUCKET"),
		S3Endpoint:  os.Getenv("GITHUT_S3_ENDPOINT"),
		AWSAccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AWSSessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
		HTTPAddr:           os.Getenv("GITHUT_HTTP_ADDR"),
		SSHAddr:            os.Getenv("GITHUT_SSH_ADDR"),
	}
}
