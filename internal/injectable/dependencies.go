package injectable

import (
	"context"
	"log"

	"github.com/bravo68web/stasis/internal/application/service"
	"github.com/bravo68web/stasis/internal/config"
	domainservice "github.com/bravo68web/stasis/internal/domain/service"
	"github.com/bravo68web/stasis/internal/infrastructure/database"
	"github.com/bravo68web/stasis/internal/infrastructure/git"
	"github.com/bravo68web/stasis/internal/infrastructure/repository"
	"github.com/bravo68web/stasis/internal/infrastructure/storage"
)

// Dependencies holds all the dependencies required by the router
type Dependencies struct {
	// Services
	AuthService   domainservice.AuthService
	GitService    domainservice.GitService
	RepoService   *service.RepoService
	UserService   *service.UserService
	SSHKeyService *service.SSHKeyService
	TokenService  *service.TokenService
	OIDCService   *service.OIDCService
	Storage       domainservice.StorageService
}

func LoadDependencies(cfg *config.Config, db *database.Database) Dependencies {
	// Initialize repositories
	userRepo := repository.NewUserRepository(db.DB())
	repoRepo := repository.NewRepoRepository(db.DB())
	sshKeyRepo := repository.NewSSHKeyRepository(db.DB())
	tokenRepo := repository.NewTokenRepository(db.DB())

	// Initialize storage
	storageFactory := storage.NewFactory(&cfg.Storage)
	storageService, err := storageFactory.Create()
	if err != nil {
		panic("Failed to initialize storage service: " + err.Error())
	}

	// Initialize OIDC service
	oidcService := service.NewOIDCService(&cfg.OIDC, userRepo)
	if cfg.OIDC.Enabled {
		if err := oidcService.Initialize(context.Background()); err != nil {
			log.Printf("Warning: Failed to initialize OIDC service: %v", err)
			// Don't panic - OIDC might be optional or provider might be temporarily unavailable
		}
	}

	// Initialize services
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

	return Dependencies{
		AuthService:   authService,
		GitService:    gitService,
		RepoService:   repoService,
		UserService:   userService,
		SSHKeyService: sshKeyService,
		TokenService:  tokenService,
		OIDCService:   oidcService,
		Storage:       storageService,
	}
}
