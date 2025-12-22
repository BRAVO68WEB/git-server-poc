package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/bravo68web/stasis/internal/domain/repository"
	"github.com/bravo68web/stasis/internal/domain/service"
	apperrors "github.com/bravo68web/stasis/pkg/errors"
	"github.com/bravo68web/stasis/pkg/logger"
)

// RepoService handles repository management operations
type RepoService struct {
	repoRepo   repository.RepoRepository
	userRepo   repository.UserRepository
	gitService service.GitService
	storage    service.StorageService
	log        *logger.Logger
}

// NewRepoService creates a new RepoService instance
func NewRepoService(
	repoRepo repository.RepoRepository,
	userRepo repository.UserRepository,
	gitService service.GitService,
	storage service.StorageService,
) *RepoService {
	return &RepoService{
		repoRepo:   repoRepo,
		userRepo:   userRepo,
		gitService: gitService,
		storage:    storage,
		log:        logger.Get().WithFields(logger.Component("repo-service")),
	}
}

// CreateRepository creates a new repository for a user
func (s *RepoService) CreateRepository(ctx context.Context, ownerID uuid.UUID, name, description string, isPrivate bool) (*models.Repository, error) {
	s.log.Info("Creating repository",
		logger.String("owner_id", ownerID.String()),
		logger.String("name", name),
		logger.Bool("is_private", isPrivate),
	)

	// Validate repository name
	if name == "" {
		s.log.Warn("Repository creation failed - name is required")
		return nil, apperrors.BadRequest("repository name is required", apperrors.ErrInvalidInput)
	}

	// Get owner to verify they exist and get username
	owner, err := s.userRepo.FindByID(ctx, ownerID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			s.log.Warn("Repository creation failed - owner not found",
				logger.String("owner_id", ownerID.String()),
			)
			return nil, apperrors.NotFound("user", apperrors.ErrNotFound)
		}
		s.log.Error("Failed to find owner",
			logger.Error(err),
			logger.String("owner_id", ownerID.String()),
		)
		return nil, fmt.Errorf("failed to find owner: %w", err)
	}

	// Check if repository already exists for this owner
	exists, err := s.repoRepo.ExistsByOwnerAndName(ctx, ownerID, name)
	if err != nil {
		s.log.Error("Failed to check repository existence",
			logger.Error(err),
			logger.String("owner_id", ownerID.String()),
			logger.String("name", name),
		)
		return nil, fmt.Errorf("failed to check repository existence: %w", err)
	}
	if exists {
		s.log.Warn("Repository already exists",
			logger.String("owner", owner.Username),
			logger.String("name", name),
		)
		return nil, apperrors.Conflict("repository already exists", apperrors.ErrRepositoryExists)
	}

	// Build git path
	gitPath := s.storage.GetRepoPath(owner.Username, name)

	// Create repository record
	repo := &models.Repository{
		Name:        name,
		OwnerID:     ownerID,
		IsPrivate:   isPrivate,
		Description: description,
		GitPath:     gitPath,
	}

	// Initialize git repository on storage
	s.log.Debug("Initializing git repository",
		logger.String("git_path", gitPath),
	)
	if err := s.gitService.InitRepository(ctx, gitPath, true); err != nil {
		s.log.Error("Failed to initialize git repository",
			logger.Error(err),
			logger.String("git_path", gitPath),
		)
		return nil, fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Save to database
	if err := s.repoRepo.Create(ctx, repo); err != nil {
		s.log.Error("Failed to create repository in database",
			logger.Error(err),
			logger.String("name", name),
		)
		// Cleanup git repository if database save fails
		if cleanupErr := s.storage.DeleteDirectory(gitPath); cleanupErr != nil {
			s.log.Error("Failed to cleanup git repository after database error",
				logger.Error(cleanupErr),
				logger.String("git_path", gitPath),
			)
		}
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	// Set owner reference
	repo.Owner = *owner

	s.log.Info("Repository created successfully",
		logger.String("repo_id", repo.ID.String()),
		logger.String("owner", owner.Username),
		logger.String("name", name),
		logger.String("git_path", gitPath),
	)

	return repo, nil
}

// GetRepository retrieves a repository by owner and name
func (s *RepoService) GetRepository(ctx context.Context, ownerUsername, repoName string) (*models.Repository, error) {
	repo, err := s.repoRepo.FindByOwnerUsernameAndName(ctx, ownerUsername, repoName)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// GetRepositoryByID retrieves a repository by its ID
func (s *RepoService) GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	return s.repoRepo.FindByID(ctx, id)
}

// ListUserRepositories lists all repositories owned by a user
func (s *RepoService) ListUserRepositories(ctx context.Context, ownerID uuid.UUID) ([]*models.Repository, error) {
	return s.repoRepo.FindByOwner(ctx, ownerID)
}

// ListPublicRepositories lists public repositories with pagination
func (s *RepoService) ListPublicRepositories(ctx context.Context, limit, offset int) ([]*models.Repository, error) {
	return s.repoRepo.ListPublic(ctx, limit, offset)
}

// UpdateRepository updates a repository's metadata
func (s *RepoService) UpdateRepository(ctx context.Context, id uuid.UUID, description *string, isPrivate *bool) (*models.Repository, error) {
	repo, err := s.repoRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if description != nil {
		repo.Description = *description
	}
	if isPrivate != nil {
		repo.IsPrivate = *isPrivate
	}

	if err := s.repoRepo.Update(ctx, repo); err != nil {
		return nil, fmt.Errorf("failed to update repository: %w", err)
	}

	return repo, nil
}

// DeleteRepository deletes a repository
func (s *RepoService) DeleteRepository(ctx context.Context, id uuid.UUID) error {
	s.log.Info("Deleting repository",
		logger.String("repo_id", id.String()),
	)

	// Get repository to get git path
	repo, err := s.repoRepo.FindByID(ctx, id)
	if err != nil {
		s.log.Error("Failed to find repository for deletion",
			logger.Error(err),
			logger.String("repo_id", id.String()),
		)
		return err
	}

	s.log.Debug("Found repository for deletion",
		logger.String("name", repo.Name),
		logger.String("git_path", repo.GitPath),
	)

	// Delete from database first
	if err := s.repoRepo.Delete(ctx, id); err != nil {
		s.log.Error("Failed to delete repository from database",
			logger.Error(err),
			logger.String("repo_id", id.String()),
		)
		return fmt.Errorf("failed to delete repository from database: %w", err)
	}

	// Delete git repository from storage
	if err := s.storage.DeleteDirectory(repo.GitPath); err != nil {
		s.log.Error("Failed to delete git repository from storage - manual cleanup may be required",
			logger.Error(err),
			logger.String("git_path", repo.GitPath),
		)
		// Don't return error since database record is already deleted
	}

	s.log.Info("Repository deleted successfully",
		logger.String("repo_id", id.String()),
		logger.String("name", repo.Name),
	)

	return nil
}

// CanUserAccessRepository checks if a user can access a repository
func (s *RepoService) CanUserAccessRepository(ctx context.Context, userID *uuid.UUID, repo *models.Repository, action string) bool {
	// Public repos allow read access to everyone
	if !repo.IsPrivate && action == "read" {
		return true
	}

	// No user means no access to private repos
	if userID == nil {
		return false
	}

	// Owner has full access
	if *userID == repo.OwnerID {
		return true
	}

	// TODO: Check collaborator permissions when implemented

	return false
}

// GetRepositoryPath returns the storage path for a repository
func (s *RepoService) GetRepositoryPath(ownerUsername, repoName string) string {
	return s.storage.GetRepoPath(ownerUsername, repoName)
}

// RepositoryExists checks if a repository exists
func (s *RepoService) RepositoryExists(ctx context.Context, ownerUsername, repoName string) (bool, error) {
	_, err := s.repoRepo.FindByOwnerUsernameAndName(ctx, ownerUsername, repoName)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CountUserRepositories returns the count of repositories for a user
func (s *RepoService) CountUserRepositories(ctx context.Context, ownerID uuid.UUID) (int64, error) {
	return s.repoRepo.CountByOwner(ctx, ownerID)
}

// ListBranches lists all branches in a repository
func (s *RepoService) ListBranches(ctx context.Context, repo *models.Repository) ([]service.Branch, error) {
	return s.gitService.ListBranches(ctx, repo.GitPath)
}

// CreateBranch creates a new branch in a repository
func (s *RepoService) CreateBranch(ctx context.Context, repo *models.Repository, branchName, commitHash string) error {
	return s.gitService.CreateBranch(ctx, repo.GitPath, branchName, commitHash)
}

// DeleteBranch deletes a branch from a repository
func (s *RepoService) DeleteBranch(ctx context.Context, repo *models.Repository, branchName string) error {
	// Check if it's the default branch
	defaultBranch, err := s.gitService.GetDefaultBranch(ctx, repo.GitPath)
	if err == nil && defaultBranch == branchName {
		return apperrors.BadRequest("cannot delete default branch", apperrors.ErrDefaultBranch)
	}

	return s.gitService.DeleteBranch(ctx, repo.GitPath, branchName)
}

// ListTags lists all tags in a repository
func (s *RepoService) ListTags(ctx context.Context, repo *models.Repository) ([]service.Tag, error) {
	return s.gitService.ListTags(ctx, repo.GitPath)
}

// CreateTag creates a new tag in a repository
func (s *RepoService) CreateTag(ctx context.Context, repo *models.Repository, tagName, commitHash, message string) error {
	return s.gitService.CreateTag(ctx, repo.GitPath, tagName, commitHash, message)
}

// DeleteTag deletes a tag from a repository
func (s *RepoService) DeleteTag(ctx context.Context, repo *models.Repository, tagName string) error {
	return s.gitService.DeleteTag(ctx, repo.GitPath, tagName)
}

// GetRepositoryStats returns statistics for a repository
func (s *RepoService) GetRepositoryStats(ctx context.Context, repo *models.Repository) (*RepositoryStats, error) {
	branches, err := s.gitService.ListBranches(ctx, repo.GitPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	tags, err := s.gitService.ListTags(ctx, repo.GitPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	// Get disk usage
	diskUsage, err := s.storage.GetDiskUsage(repo.GitPath)
	if err != nil {
		diskUsage = 0
	}

	return &RepositoryStats{
		BranchCount: len(branches),
		TagCount:    len(tags),
		DiskUsage:   diskUsage,
	}, nil
}

// RepositoryStats holds statistics for a repository
type RepositoryStats struct {
	BranchCount int   `json:"branch_count"`
	TagCount    int   `json:"tag_count"`
	DiskUsage   int64 `json:"disk_usage"`
}

// TransferRepository transfers a repository to a new owner
func (s *RepoService) TransferRepository(ctx context.Context, repoID, newOwnerID uuid.UUID) (*models.Repository, error) {
	s.log.Info("Transferring repository",
		logger.String("repo_id", repoID.String()),
		logger.String("new_owner_id", newOwnerID.String()),
	)

	// Get repository
	repo, err := s.repoRepo.FindByID(ctx, repoID)
	if err != nil {
		s.log.Error("Failed to find repository for transfer",
			logger.Error(err),
			logger.String("repo_id", repoID.String()),
		)
		return nil, err
	}

	// Get new owner
	newOwner, err := s.userRepo.FindByID(ctx, newOwnerID)
	if err != nil {
		s.log.Error("Failed to find new owner for transfer",
			logger.Error(err),
			logger.String("new_owner_id", newOwnerID.String()),
		)
		return nil, fmt.Errorf("failed to find new owner: %w", err)
	}

	// Check if new owner already has a repo with this name
	exists, err := s.repoRepo.ExistsByOwnerAndName(ctx, newOwnerID, repo.Name)
	if err != nil {
		s.log.Error("Failed to check repository existence for transfer",
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to check repository existence: %w", err)
	}
	if exists {
		s.log.Warn("Transfer failed - new owner already has repository with same name",
			logger.String("new_owner", newOwner.Username),
			logger.String("repo_name", repo.Name),
		)
		return nil, apperrors.Conflict("new owner already has a repository with this name", apperrors.ErrRepositoryExists)
	}

	// Get old and new paths
	oldPath := repo.GitPath
	newPath := s.storage.GetRepoPath(newOwner.Username, repo.Name)

	// Move git repository
	s.log.Debug("Moving git repository",
		logger.String("old_path", oldPath),
		logger.String("new_path", newPath),
	)
	if err := s.storage.MoveFile(oldPath, newPath); err != nil {
		s.log.Error("Failed to move git repository",
			logger.Error(err),
			logger.String("old_path", oldPath),
			logger.String("new_path", newPath),
		)
		return nil, fmt.Errorf("failed to move repository: %w", err)
	}

	// Update database record
	repo.OwnerID = newOwnerID
	repo.GitPath = newPath

	if err := s.repoRepo.Update(ctx, repo); err != nil {
		s.log.Error("Failed to update repository after move",
			logger.Error(err),
		)
		// Try to move back on failure
		if moveErr := s.storage.MoveFile(newPath, oldPath); moveErr != nil {
			s.log.Error("Failed to rollback repository move",
				logger.Error(moveErr),
				logger.String("new_path", newPath),
				logger.String("old_path", oldPath),
			)
		}
		return nil, fmt.Errorf("failed to update repository: %w", err)
	}

	repo.Owner = *newOwner

	s.log.Info("Repository transferred successfully",
		logger.String("repo_id", repoID.String()),
		logger.String("repo_name", repo.Name),
		logger.String("new_owner", newOwner.Username),
	)

	return repo, nil
}

// GetCommits returns a list of commits for a repository
func (s *RepoService) GetCommits(ctx context.Context, repo *models.Repository, ref string, limit, offset int) ([]service.Commit, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return s.gitService.GetCommits(ctx, repo.GitPath, ref, limit, offset)
}

// GetCommit returns a single commit by hash
func (s *RepoService) GetCommit(ctx context.Context, repo *models.Repository, commitHash string) (*service.Commit, error) {
	return s.gitService.GetCommit(ctx, repo.GitPath, commitHash)
}

// GetTree returns the tree entries for a repository at a given ref and path
func (s *RepoService) GetTree(ctx context.Context, repo *models.Repository, ref, path string) ([]service.TreeEntry, error) {
	return s.gitService.GetTree(ctx, repo.GitPath, ref, path)
}

// GetFileContent returns the content of a file in a repository
func (s *RepoService) GetFileContent(ctx context.Context, repo *models.Repository, ref, filePath string) (*service.FileContent, error) {
	return s.gitService.GetFileContent(ctx, repo.GitPath, ref, filePath)
}

// GetBlame returns blame information for a file in a repository
func (s *RepoService) GetBlame(ctx context.Context, repo *models.Repository, ref, filePath string) ([]service.BlameLine, error) {
	return s.gitService.GetBlame(ctx, repo.GitPath, ref, filePath)
}

// GetDiff returns the diff (patch) for a specific commit in a repository
func (s *RepoService) GetDiff(ctx context.Context, repo *models.Repository, commitHash string) (*service.DiffResult, error) {
	return s.gitService.GetDiff(ctx, repo.GitPath, commitHash)
}

// ForkRepository creates a fork of a repository
func (s *RepoService) ForkRepository(ctx context.Context, sourceRepoID, newOwnerID uuid.UUID, newName string) (*models.Repository, error) {
	s.log.Info("Forking repository",
		logger.String("source_repo_id", sourceRepoID.String()),
		logger.String("new_owner_id", newOwnerID.String()),
		logger.String("new_name", newName),
	)

	// Get source repository
	sourceRepo, err := s.repoRepo.FindByID(ctx, sourceRepoID)
	if err != nil {
		s.log.Error("Failed to find source repository for fork",
			logger.Error(err),
			logger.String("source_repo_id", sourceRepoID.String()),
		)
		return nil, err
	}

	// If no new name provided, use source name
	if newName == "" {
		newName = sourceRepo.Name
	}

	// Get new owner
	newOwner, err := s.userRepo.FindByID(ctx, newOwnerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find new owner: %w", err)
	}

	// Check if new owner already has a repo with this name
	exists, err := s.repoRepo.ExistsByOwnerAndName(ctx, newOwnerID, newName)
	if err != nil {
		s.log.Error("Failed to check repository existence for fork",
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to check repository existence: %w", err)
	}
	if exists {
		s.log.Warn("Fork failed - user already has repository with same name",
			logger.String("new_owner", newOwner.Username),
			logger.String("repo_name", newName),
		)
		return nil, apperrors.Conflict("you already have a repository with this name", apperrors.ErrRepositoryExists)
	}

	// Build new git path
	newGitPath := s.storage.GetRepoPath(newOwner.Username, newName)

	// Clone the repository
	s.log.Debug("Cloning repository for fork",
		logger.String("source_path", sourceRepo.GitPath),
		logger.String("new_path", newGitPath),
	)
	if err := s.gitService.CloneRepository(ctx, sourceRepo.GitPath, newGitPath, true); err != nil {
		s.log.Error("Failed to clone repository for fork",
			logger.Error(err),
			logger.String("source_path", sourceRepo.GitPath),
			logger.String("new_path", newGitPath),
		)
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Create repository record
	newRepo := &models.Repository{
		Name:        newName,
		OwnerID:     newOwnerID,
		IsPrivate:   sourceRepo.IsPrivate,
		Description: fmt.Sprintf("Fork of %s/%s", sourceRepo.Owner.Username, sourceRepo.Name),
		GitPath:     newGitPath,
	}

	if err := s.repoRepo.Create(ctx, newRepo); err != nil {
		s.log.Error("Failed to create forked repository in database",
			logger.Error(err),
		)
		// Cleanup on failure
		if cleanupErr := s.storage.DeleteDirectory(newGitPath); cleanupErr != nil {
			s.log.Error("Failed to cleanup forked repository after database error",
				logger.Error(cleanupErr),
				logger.String("git_path", newGitPath),
			)
		}
		return nil, fmt.Errorf("failed to create forked repository: %w", err)
	}

	newRepo.Owner = *newOwner

	s.log.Info("Repository forked successfully",
		logger.String("source_repo", fmt.Sprintf("%s/%s", sourceRepo.Owner.Username, sourceRepo.Name)),
		logger.String("new_repo", fmt.Sprintf("%s/%s", newOwner.Username, newName)),
		logger.String("new_repo_id", newRepo.ID.String()),
	)

	return newRepo, nil
}
