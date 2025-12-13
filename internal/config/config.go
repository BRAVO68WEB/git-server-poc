package config

import (
	"os"
)

type Config struct {
	PostgresDSN string
	S3Region    string
	S3Bucket    string
	S3Endpoint  string
	HTTPAddr    string
	SSHAddr     string
}

func Load() Config {
	return Config{
		PostgresDSN: os.Getenv("GITHUT_POSTGRES_DSN"),
		S3Region:    os.Getenv("GITHUT_S3_REGION"),
		S3Bucket:    os.Getenv("GITHUT_S3_BUCKET"),
		S3Endpoint:  os.Getenv("GITHUT_S3_ENDPOINT"),
		HTTPAddr:    os.Getenv("GITHUT_HTTP_ADDR"),
		SSHAddr:     os.Getenv("GITHUT_SSH_ADDR"),
	}
}
