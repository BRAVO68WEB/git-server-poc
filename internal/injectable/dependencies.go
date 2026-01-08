package injectable

import (
	"context"

	"time"

	"github.com/bravo68web/stasis/internal/application/service"
	"github.com/bravo68web/stasis/internal/config"
	domainservice "github.com/bravo68web/stasis/internal/domain/service"
	"github.com/bravo68web/stasis/internal/infrastructure/database"
	"github.com/bravo68web/stasis/internal/infrastructure/git"
	"github.com/bravo68web/stasis/internal/infrastructure/repository"
	"github.com/bravo68web/stasis/internal/infrastructure/storage"
	"github.com/bravo68web/stasis/pkg/logger"
)

// Dependencies holds all the dependencies required by the router
type Dependencies struct {
	// Services
	AuthService       domainservice.AuthService
	GitService        domainservice.GitService
	RepoService       *service.RepoService
	UserService       *service.UserService
	SSHKeyService     *service.SSHKeyService
	TokenService      *service.TokenService
	OIDCService       *service.OIDCService
	CIService         *service.CIService
	MirrorSyncService *service.MirrorSyncService
	MirrorCronService *service.MirrorCronService
	Storage           domainservice.StorageService
}

func LoadDependencies(cfg *config.Config, db *database.Database) Dependencies {
	log := logger.Get()

	log.Info("Loading application dependencies...")

	// Initialize repositories
	log.Debug("Initializing repositories...")
	userRepo := repository.NewUserRepository(db.DB())
	repoRepo := repository.NewRepoRepository(db.DB())
	sshKeyRepo := repository.NewSSHKeyRepository(db.DB())
	tokenRepo := repository.NewTokenRepository(db.DB())

	log.Debug("Repositories initialized",
		logger.Int("count", 4),
	)

	// Initialize storage
	log.Debug("Initializing storage service...",
		logger.String("type", cfg.Storage.Type),
	)
	storageFactory := storage.NewFactory(&cfg.Storage)
	storageService, err := storageFactory.Create()
	if err != nil {
		log.Fatal("Failed to initialize storage service",
			logger.Error(err),
			logger.String("storage_type", cfg.Storage.Type),
		)
	}
	log.Info("Storage service initialized",
		logger.String("type", cfg.Storage.Type),
		logger.String("base_path", cfg.Storage.BasePath),
	)

	// Initialize OIDC service
	log.Debug("Initializing OIDC service...",
		logger.Bool("enabled", cfg.OIDC.Enabled),
	)
	oidcService := service.NewOIDCService(&cfg.OIDC, userRepo)
	if cfg.OIDC.Enabled {
		if err := oidcService.Initialize(context.Background()); err != nil {
			log.Warn("Failed to initialize OIDC service - OIDC authentication will be unavailable",
				logger.Error(err),
				logger.String("issuer_url", cfg.OIDC.IssuerURL),
			)
			// Don't panic - OIDC might be optional or provider might be temporarily unavailable
		} else {
			log.Info("OIDC service initialized successfully",
				logger.String("issuer_url", cfg.OIDC.IssuerURL),
			)
		}
	} else {
		log.Info("OIDC service is disabled")
	}

	// Initialize services
	log.Debug("Initializing application services...")
	authService := service.NewAuthService(userRepo, sshKeyRepo, tokenRepo, oidcService, &cfg.OIDC)
	gitService := git.NewGitOperations(storageService)
	repoService := service.NewRepoService(
		repoRepo,
		userRepo,
		gitService,
		storageService,
	)
	userService := service.NewUserService(userRepo)
	sshKeyService := service.NewSSHKeyService(sshKeyRepo, userRepo)
	tokenService := service.NewTokenService(tokenRepo, userRepo)

	// Initialize CI service
	// CI data (jobs, logs, artifacts) is fetched directly from CI server - no local database storage
	log.Debug("Initializing CI service...",
		logger.Bool("enabled", cfg.CI.Enabled),
	)
	ciService := service.NewCIService(
		&cfg.CI,
		repoRepo,
	)
	if cfg.CI.Enabled {
		log.Info("CI service initialized successfully (fetching from CI server)",
			logger.String("server_url", cfg.CI.ServerURL),
		)
	} else {
		log.Info("CI service is disabled")
	}

	// Initialize mirror sync services
	log.Debug("Initializing mirror sync services...")
	mirrorSyncService := service.NewMirrorSyncService(
		repoRepo,
		gitService,
	)

	// Initialize mirror cron service (checks per-repository intervals)
	mirrorCronService := service.NewMirrorCronService(
		mirrorSyncService,
		1*time.Minute, // Check every minute for repositories that need syncing
	)

	// Always start cron service (it will check each repo's individual settings)
	mirrorCronService.Start()
	log.Info("Mirror sync cron service started (per-repository intervals)")

	log.Info("All application services initialized successfully",
		logger.Bool("auth_service", true),
		logger.Bool("git_service", true),
		logger.Bool("repo_service", true),
		logger.Bool("user_service", true),
		logger.Bool("ssh_key_service", true),
		logger.Bool("token_service", true),
		logger.Bool("oidc_service", cfg.OIDC.Enabled),
		logger.Bool("ci_service", cfg.CI.Enabled),
		logger.Bool("mirror_sync_service", true),
		logger.Bool("mirror_cron_service", true),
	)

	log.Info("Dependencies loaded successfully")

	return Dependencies{
		AuthService:       authService,
		GitService:        gitService,
		RepoService:       repoService,
		UserService:       userService,
		SSHKeyService:     sshKeyService,
		TokenService:      tokenService,
		OIDCService:       oidcService,
		CIService:         ciService,
		MirrorSyncService: mirrorSyncService,
		MirrorCronService: mirrorCronService,
		Storage:           storageService,
	}
}
