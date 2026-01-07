package service

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"

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

// GetRepoRepository returns the underlying repository for CI integration
func (s *RepoService) GetRepoRepository() repository.RepoRepository {
	return s.repoRepo
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
func (s *RepoService) UpdateRepository(ctx context.Context, id uuid.UUID, description *string, isPrivate *bool, defaultBranch *string) (*models.Repository, error) {
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
	if defaultBranch != nil {
		// Verify the branch exists before updating
		exists, err := s.gitService.BranchExists(ctx, repo.GitPath, *defaultBranch)
		if err != nil {
			s.log.Error("Failed to check if branch exists",
				logger.Error(err),
				logger.String("branch", *defaultBranch),
			)
			return nil, fmt.Errorf("failed to verify branch: %w", err)
		}
		if !exists {
			return nil, apperrors.BadRequest("branch does not exist", apperrors.ErrInvalidInput)
		}

		// Update git HEAD to point to the new branch
		if err := s.gitService.SetHEADBranch(ctx, repo.GitPath, *defaultBranch); err != nil {
			s.log.Error("Failed to set default branch in git",
				logger.Error(err),
				logger.String("branch", *defaultBranch),
			)
			return nil, fmt.Errorf("failed to set default branch: %w", err)
		}

		repo.DefaultBranch = *defaultBranch
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
	defaultBranch, err := s.gitService.GetHEADBranch(ctx, repo.GitPath)
	if err == nil && defaultBranch == branchName {
		return apperrors.BadRequest("cannot delete default branch", apperrors.ErrDefaultBranch)
	}

	return s.gitService.DeleteBranch(ctx, repo.GitPath, branchName)
}

// SetDefaultBranchOnPush sets the default branch for a repository after a push
// This is called after a successful push to set the default branch if it's not already set
func (s *RepoService) SetDefaultBranchOnPush(ctx context.Context, repo *models.Repository) error {
	// Check if default branch is already set and valid
	if repo.DefaultBranch != "" {
		// Verify the current default branch still exists
		exists, err := s.gitService.BranchExists(ctx, repo.GitPath, repo.DefaultBranch)
		if err == nil && exists {
			// Default branch is already set and valid, nothing to do
			return nil
		}
	}

	// Get the current HEAD from git (which should point to the pushed branch)
	defaultBranch, err := s.gitService.GetHEADBranch(ctx, repo.GitPath)
	if err != nil {
		s.log.Warn("Failed to get default branch from git",
			logger.Error(err),
			logger.String("repo_id", repo.ID.String()),
		)
		// Fall through to try finding any branch
		defaultBranch = ""
	}

	// Check if the default branch from HEAD actually exists
	if defaultBranch != "" {
		exists, err := s.gitService.BranchExists(ctx, repo.GitPath, defaultBranch)
		if err != nil || !exists {
			s.log.Debug("HEAD points to non-existent branch",
				logger.String("repo_id", repo.ID.String()),
				logger.String("branch", defaultBranch),
			)
			defaultBranch = ""
		}
	}

	if defaultBranch == "" {
		// Try to find any branch
		branches, err := s.gitService.ListBranches(ctx, repo.GitPath)
		if err != nil || len(branches) == 0 {
			s.log.Debug("No branches found in repository",
				logger.String("repo_id", repo.ID.String()),
			)
			return nil
		}
		defaultBranch = branches[0].Name

		// Update git HEAD to point to the actual branch
		if err := s.gitService.SetHEADBranch(ctx, repo.GitPath, defaultBranch); err != nil {
			s.log.Error("Failed to set default branch in git",
				logger.Error(err),
				logger.String("repo_id", repo.ID.String()),
				logger.String("branch", defaultBranch),
			)
			// Continue anyway to update the database
		}
	}

	// Update the repository record
	repo.DefaultBranch = defaultBranch
	if err := s.repoRepo.Update(ctx, repo); err != nil {
		s.log.Error("Failed to update default branch in database",
			logger.Error(err),
			logger.String("repo_id", repo.ID.String()),
			logger.String("branch", defaultBranch),
		)
		return nil // Don't fail the push for this
	}

	s.log.Info("Default branch set after push",
		logger.String("repo_id", repo.ID.String()),
		logger.String("branch", defaultBranch),
	)

	return nil
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

	// Calculate total commit count
	var totalCommits int
	for _, branch := range branches {
		totalCommits += branch.CommitCount
	}

	// Language Usage Percentage
	languageUsagePerc := s.calculateLanguageUsagePercentage(ctx, repo)

	return &RepositoryStats{
		BranchCount:       len(branches),
		TagCount:          len(tags),
		DiskUsage:         diskUsage,
		TotalCommits:      totalCommits,
		LanguageUsagePerc: languageUsagePerc,
	}, nil
}

var extensionToLanguage = map[string]string{
	".go":         "Go",
	".js":         "JavaScript",
	".ts":         "TypeScript",
	".tsx":        "TypeScript",
	".jsx":        "JavaScript",
	".py":         "Python",
	".java":       "Java",
	".c":          "C",
	".cpp":        "C++",
	".h":          "C/C++",
	".cs":         "C#",
	".rb":         "Ruby",
	".php":        "PHP",
	".html":       "HTML",
	".css":        "CSS",
	".scss":       "CSS",
	".json":       "JSON",
	".xml":        "XML",
	".yaml":       "YAML",
	".yml":        "YAML",
	".md":         "Markdown",
	".sql":        "SQL",
	".sh":         "Shell",
	".bash":       "Shell",
	".rs":         "Rust",
	".swift":      "Swift",
	".kt":         "Kotlin",
	".dart":       "Dart",
	".lua":        "Lua",
	".dockerfile": "Dockerfile",
}

var ignoredDirs = map[string]struct{}{
	"vendor":       {},
	"node_modules": {},
	".git":         {},
	".github":      {},
	"dist":         {},
	"build":        {},
	"out":          {},
	"coverage":     {},
}

var ignoredFileExts = map[string]struct{}{
	".png":   {},
	".jpg":   {},
	".jpeg":  {},
	".gif":   {},
	".svg":   {},
	".ico":   {},
	".pdf":   {},
	".zip":   {},
	".tar":   {},
	".gz":    {},
	".rar":   {},
	".7z":    {},
	".woff":  {},
	".woff2": {},
	".ttf":   {},
	".eot":   {},
	".mp3":   {},
	".mp4":   {},
	".mov":   {},
	".avi":   {},
	".bin":   {},
	".exe":   {},
	".dll":   {},
	".so":    {},
	".dylib": {},
}

var ignoredFilenames = map[string]struct{}{
	"package-lock.json": {},
	"yarn.lock":         {},
	"pnpm-lock.yaml":    {},
	"Cargo.lock":        {},
}

func (s *RepoService) calculateLanguageUsagePercentage(ctx context.Context, repo *models.Repository) map[string]float64 {
	refHash, err := s.gitService.GetHEADRef(ctx, repo.GitPath)
	if err != nil {
		return map[string]float64{}
	}
	ref := refHash
	if ref == "" {
		if branch, _ := s.gitService.GetHEADBranch(ctx, repo.GitPath); branch != "" {
			ref = branch
		} else {
			if branches, _ := s.gitService.ListBranches(ctx, repo.GitPath); len(branches) > 0 {
				ref = branches[0].Name
			} else {
				return map[string]float64{}
			}
		}
	}

	langSizes := make(map[string]int64)
	var totalSize int64

	var walk func(path string) error
	walk = func(path string) error {
		entries, err := s.gitService.GetTree(ctx, repo.GitPath, ref, path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if entry.Type == "blob" {
				if entry.Mode == "120000" {
					continue
				}

				ext := strings.ToLower(filepath.Ext(entry.Name))
				if strings.EqualFold(entry.Name, "Dockerfile") {
					ext = ".dockerfile"
				}
				if _, ok := ignoredFileExts[ext]; ok {
					continue
				}
				if _, ok := ignoredFilenames[entry.Name]; ok {
					continue
				}

				if lang, ok := extensionToLanguage[ext]; ok {
					langSizes[lang] += entry.Size
					totalSize += entry.Size
				} else {
					langSizes["Other"] += entry.Size
					totalSize += entry.Size
				}
			} else if entry.Type == "tree" {
				if _, ok := ignoredDirs[entry.Name]; ok {
					continue
				}
				if err := walk(entry.Path); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := walk(""); err != nil {
		s.log.Error("Failed to walk tree for stats",
			logger.String("repo_path", repo.GitPath),
			logger.Error(err),
		)
		return map[string]float64{}
	}

	// Calculate percentages
	percentages := make(map[string]float64)
	if totalSize == 0 {
		return percentages
	}

	for lang, size := range langSizes {
		v := (float64(size) / float64(totalSize)) * 100
		percentages[lang] = math.Round(v*100) / 100
	}

	return percentages
}

// RepositoryStats holds statistics for a repository
type RepositoryStats struct {
	BranchCount       int                `json:"branch_count"`
	TagCount          int                `json:"tag_count"`
	DiskUsage         int64              `json:"disk_usage"`
	TotalCommits      int                `json:"total_commits"`
	LanguageUsagePerc map[string]float64 `json:"language_usage_perc"`
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

// GetCompareDiff returns the diff between two commits
func (s *RepoService) GetCompareDiff(ctx context.Context, repo *models.Repository, from, to string) (*service.DiffResult, error) {
	return s.gitService.GetCompareDiff(ctx, repo.GitPath, from, to)
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
