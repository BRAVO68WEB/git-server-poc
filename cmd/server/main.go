package main

import (
	"github.com/bravo68web/githut/internal/application/service"
	"github.com/bravo68web/githut/internal/config"
	domainservice "github.com/bravo68web/githut/internal/domain/service"
	"github.com/bravo68web/githut/internal/infrastructure/database"
	"github.com/bravo68web/githut/internal/infrastructure/git"
	"github.com/bravo68web/githut/internal/infrastructure/repository"
	"github.com/bravo68web/githut/internal/infrastructure/storage"
)

// Dependencies holds all the dependencies required by the router
type Dependencies struct {
	// Services
	AuthService domainservice.AuthService
	GitService  domainservice.GitService
	RepoService *service.RepoService
	UserService *service.UserService
	Storage     domainservice.StorageService
}

func loadDependencies(cfg *config.Config, db *database.Database) Dependencies {
	// Initialize repositories
	userRepo := repository.NewUserRepository(db.DB())
	repoRepo := repository.NewRepoRepository(db.DB())

	// Initialize storage
	storageFactory := storage.NewFactory(&cfg.Storage)
	storageService, err := storageFactory.Create()
	if err != nil {
		panic("Failed to initialize storage service: " + err.Error())
	}

	// Initialize services
	authService := service.NewAuthService(userRepo)
	gitService := git.NewGitOperations(storageService)
	repoService := service.NewRepoService(
		repoRepo,
		userRepo,
		gitService,
		storageService,
	)
	userService := service.NewUserService(userRepo, authService)

	return Dependencies{
		AuthService: authService,
		GitService:  gitService,
		RepoService: repoService,
		UserService: userService,
		Storage:     storageService,
	}
}

func main() {
}
