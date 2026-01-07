package server

import (
	"context"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap/zapcore"

	"github.com/bravo68web/stasis/internal/config"
	"github.com/bravo68web/stasis/internal/infrastructure/database"
	"github.com/bravo68web/stasis/internal/infrastructure/otel"
	"github.com/bravo68web/stasis/pkg/logger"
	"github.com/bravo68web/stasis/pkg/openapi"
)

type Server struct {
	*gin.Engine
	OpenAPIGenerator *openapi.Generator

	Config       *config.Config
	DB           *database.Database
	Logger       *logger.Logger
	OTELProvider *otel.Provider
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

	// Initialize logger based on configuration
	loggerCfg := cfg.Logging.ToLoggerConfig()
	var log *logger.Logger
	var otelProvider *otel.Provider

	// Check if OTEL output is configured
	if strings.ToLower(cfg.Logging.Output) == "otel" && cfg.Logging.OTEL.Enabled {
		// Initialize OTEL provider
		otelCfg := cfg.Logging.OTEL.ToOTELConfig()
		otelProvider, err = otel.NewProvider(otelCfg)
		if err != nil {
			// Fall back to console logging if OTEL fails
			loggerCfg.Output = logger.OutputConsole
			log, err = logger.New(loggerCfg)
			if err != nil {
				panic("Failed to initialize logger: " + err.Error())
			}
			log.Warn("Failed to initialize OTEL provider, falling back to console logging",
				logger.Error(err),
			)
		} else {
			// Create logger with OTEL core
			level, _ := parseLevel(cfg.Logging.Level)

			// Create base console core for local logging
			baseLog, err := logger.New(loggerCfg)
			if err != nil {
				panic("Failed to initialize base logger: " + err.Error())
			}

			// Create combined core that writes to both console and OTEL
			combinedCore := otel.NewCombinedCore(baseLog.Core(), otelProvider, level)
			log = logger.NewWithCore(loggerCfg, combinedCore, otelProvider)
		}
	} else {
		// Initialize standard logger (console or file)
		log, err = logger.New(loggerCfg)
		if err != nil {
			panic("Failed to initialize logger: " + err.Error())
		}
	}

	// Set global logger
	logger.SetGlobal(log)

	log.Info("Logger initialized",
		logger.String("level", cfg.Logging.Level),
		logger.String("output", cfg.Logging.Output),
		logger.String("format", cfg.Logging.Format),
	)

	// Initialize database connection
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		log.Fatal("Failed to initialize database", logger.Error(err))
	}

	log.Info("Database connection established",
		logger.String("host", cfg.Database.Host),
		logger.Int("port", cfg.Database.Port),
		logger.String("database", cfg.Database.DBName),
	)

	// Run auto-migrations in production mode
	if cfg.IsProduction() {
		migrator := database.NewMigrator(db)
		migrator.MustApplyMigrations(context.Background())

		log.Info("Database migrations applied")
	}

	// Set Gin mode based on configuration
	switch cfg.Server.Mode {
	case "release":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	// Create Gin engine without default middleware
	engine := gin.New()

	apiGen := openapi.NewGenerator(engine, openapi.Info{
		Title:       "Stasis - Git Server API",
		Version:     "1.0.0",
		Description: "A self-hosted Git server API that provides repository management, authentication, SSH key management, and Git Smart HTTP protocol support.",
		Contact: &openapi.Contact{
			Name: "Bravo68Web",
		},
		License: &openapi.License{
			Name: "MIT",
		},
	}, []openapi.Server{
		{URL: "http://localhost:8080", Description: "Local development server"},
	}, []openapi.Tag{
		{Name: "Health", Description: "Health check endpoints"},
		{Name: "Authentication", Description: "User authentication and registration"},
		{Name: "SSH Keys", Description: "SSH key management for Git SSH access"},
		{Name: "Repositories", Description: "Repository management operations"},
		{Name: "Branches", Description: "Branch management operations"},
		{Name: "Tags", Description: "Tag management operations"},
		{Name: "Commits", Description: "Commit history and details"},
		{Name: "Code", Description: "File tree, content, and blame information"},
		{Name: "Git Protocol", Description: "Git Smart HTTP protocol endpoints"},
	})

	log.Info("Server initialized",
		logger.String("host", cfg.Server.Host),
		logger.Int("port", cfg.Server.Port),
		logger.String("mode", cfg.Server.Mode),
	)

	return &Server{
		Engine:           engine,
		OpenAPIGenerator: apiGen,
		Config:           cfg,
		DB:               db,
		Logger:           log,
		OTELProvider:     otelProvider,
	}
}

// Close gracefully shuts down the server and its resources
func (s *Server) Close() error {
	if s.Logger != nil {
		s.Logger.Info("Shutting down server...")
	}

	// Close OTEL provider first to flush any pending logs
	if s.OTELProvider != nil {
		if err := s.OTELProvider.Close(); err != nil {
			if s.Logger != nil {
				s.Logger.Warn("Error closing OTEL provider", logger.Error(err))
			}
		}
	}

	// Close logger
	if s.Logger != nil {
		return s.Logger.Close()
	}

	return nil
}

// parseLevel converts a string level to zapcore.Level
func parseLevel(level string) (zapcore.Level, error) {
	var l zapcore.Level
	err := l.UnmarshalText([]byte(level))
	return l, err
}
