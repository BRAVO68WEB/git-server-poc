package server

import (
	"os"

	"github.com/gin-gonic/gin"

	"github.com/bravo68web/stasis/internal/config"
	"github.com/bravo68web/stasis/internal/infrastructure/database"
)

type Server struct {
	*gin.Engine

	Config *config.Config
	DB     *database.Database
}

func New() *Server {
	// Get config path from environment or use default
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize database connection
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		panic(err)
	}

	// Set Gin mode based on configuration
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	return &Server{
		Engine: gin.Default(),
		Config: cfg,
		DB:     db,
	}
}
