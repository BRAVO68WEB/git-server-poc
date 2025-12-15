package server

import (
	"os"

	"github.com/gin-gonic/gin"

	"github.com/bravo68web/githut/configs"
	"github.com/bravo68web/githut/internal/config"
	"github.com/bravo68web/githut/internal/domain/service"
	"github.com/bravo68web/githut/internal/infrastructure/database"
	"github.com/bravo68web/githut/internal/infrastructure/storage"
)

type Server struct {
	*gin.Engine

	Config         *config.Config
	DB             *database.Database
	StorageService service.StorageService
}

func New() *Server {
	// Get config path from environment or use default
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	// Load configuration (with embedded config fallback)
	cfg, err := config.LoadWithEmbedded(configPath, configs.EmbeddedConfigs)
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize database connection
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		panic(err)
	}

	// Initialize storage
	storageFactory := storage.NewFactory(&cfg.Storage)
	storageService, err := storageFactory.Create()
	if err != nil {
		panic("Failed to initialize storage service: " + err.Error())
	}

	// Set Gin mode based on configuration
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	return &Server{
		Engine:         gin.Default(),
		Config:         cfg,
		DB:             db,
		StorageService: storageService,
	}
}
