package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bravo68web/stasis/internal/application/service"
	"github.com/bravo68web/stasis/internal/domain/models"
	domainservice "github.com/bravo68web/stasis/internal/domain/service"
	"github.com/bravo68web/stasis/internal/infrastructure/git"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
	"github.com/bravo68web/stasis/pkg/logger"
	"github.com/gin-gonic/gin"
)

// GitHandler handles Git smart HTTP protocol requests
type GitHandler struct {
	gitService  domainservice.GitService
	repoService *service.RepoService
	authService domainservice.AuthService
	storage     domainservice.StorageService
	ciService   *service.CIService
	gitProtocol *git.GitProtocol
	log         *logger.Logger
}

// NewGitHandler creates a new GitHandler instance
func NewGitHandler(
	gitService domainservice.GitService,
	repoService *service.RepoService,
	authService domainservice.AuthService,
	storage domainservice.StorageService,
	ciService *service.CIService,
) *GitHandler {
	return &GitHandler{
		gitService:  gitService,
		repoService: repoService,
		authService: authService,
		storage:     storage,
		ciService:   ciService,
		gitProtocol: git.NewGitProtocol(),
		log:         logger.Get().WithFields(logger.Component("git-handler")),
	}
}

// HandleInfoRefs handles GET /{owner}/{repo}/info/refs?service=git-upload-pack|git-receive-pack
func (h *GitHandler) HandleInfoRefs(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	serviceName := c.Query("service")

	// Clean repo name (remove .git suffix if present)
	repoName = strings.TrimSuffix(repoName, ".git")

	// Validate service
	if !git.IsValidService(serviceName) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Invalid service",
		})
		return
	}

	// Get repository
	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Repository not found",
		})
		return
	}

	// Check authorization
	user := middleware.GetUserFromContext(c)
	isWriteOperation := serviceName == "git-receive-pack"

	if !h.checkRepoAccess(c, user, repo, isWriteOperation) {
		return
	}

	// Get info/refs from git
	service := git.NormalizeServiceName(serviceName)
	response, err := h.gitProtocol.GetInfoRefs(c.Request.Context(), git.InfoRefsRequest{
		RepoPath: repo.GitPath,
		Service:  service,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get repository info",
		})
		return
	}

	// Set response headers
	c.Header("Content-Type", response.ContentType)
	c.Header("Cache-Control", "no-cache")
	c.Header("Pragma", "no-cache")

	// Write response
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write(response.Body)
}

// HandleUploadPack handles POST /{owner}/{repo}/git-upload-pack (fetch/clone)
func (h *GitHandler) HandleUploadPack(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	// Clean repo name
	repoName = strings.TrimSuffix(repoName, ".git")

	// Get repository
	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Repository not found",
		})
		return
	}

	// Check authorization (read access)
	user := middleware.GetUserFromContext(c)
	if !h.checkRepoAccess(c, user, repo, false) {
		return
	}

	// Validate content type
	contentType := c.ContentType()
	if contentType != "application/x-git-upload-pack-request" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid content type",
		})
		return
	}

	// Set response headers
	c.Header("Content-Type", "application/x-git-upload-pack-result")
	c.Header("Cache-Control", "no-cache")

	// Handle upload-pack
	if err := h.gitProtocol.HandleUploadPack(c.Request.Context(), repo.GitPath, c.Request.Body, c.Writer); err != nil {
		// Response already started, can't send error JSON
		return
	}
}

// HandleReceivePack handles POST /{owner}/{repo}/git-receive-pack (push)
func (h *GitHandler) HandleReceivePack(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	// Clean repo name
	repoName = strings.TrimSuffix(repoName, ".git")

	// Get repository
	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Repository not found",
		})
		return
	}

	// Check authorization (write access required)
	user := middleware.GetUserFromContext(c)
	if !h.checkRepoAccess(c, user, repo, true) {
		return
	}

	// Validate content type
	contentType := c.ContentType()
	if contentType != "application/x-git-receive-pack-request" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid content type",
		})
		return
	}

	// Set response headers
	c.Header("Content-Type", "application/x-git-receive-pack-result")
	c.Header("Cache-Control", "no-cache")

	// Handle receive-pack
	if err := h.gitProtocol.HandleReceivePack(c.Request.Context(), repo.GitPath, c.Request.Body, c.Writer); err != nil {
		// Response already started, can't send error JSON
		return
	}

	// Sync to remote storage (S3) after successful push
	// This runs synchronously to ensure data is persisted before returning
	if err := h.storage.SyncToRemote(repo.GitPath); err != nil {
		h.log.Error("Failed to sync repository to remote storage",
			logger.Error(err),
			logger.String("repo", repo.Name),
			logger.String("path", repo.GitPath),
		)
		// Don't fail the push, just log the error
	}

	// Set default branch if not already set (first push)
	h.repoService.SetDefaultBranchOnPush(c.Request.Context(), repo)

	// Trigger CI after successful push (runs asynchronously)
	h.triggerCIAfterPush(c.Request.Context(), repo, user, owner, repoName)
}

// triggerCIAfterPush triggers CI jobs after a successful push
func (h *GitHandler) triggerCIAfterPush(ctx context.Context, repo *models.Repository, user *models.User, owner, repoName string) {
	// Check if CI service is enabled
	if h.ciService == nil || !h.ciService.IsEnabled() {
		h.log.Debug("CI service is not enabled, skipping CI trigger",
			logger.String("repo", fmt.Sprintf("%s/%s", owner, repoName)),
		)
		return
	}

	// Get the default branch (or HEAD)
	defaultBranch, err := h.gitService.GetHEADBranch(ctx, repo.GitPath)
	if err != nil {
		h.log.Warn("Failed to get default branch for CI trigger",
			logger.Error(err),
			logger.String("repo", fmt.Sprintf("%s/%s", owner, repoName)),
		)
		return
	}

	// Get the latest commit on the default branch
	commits, err := h.gitService.GetCommits(ctx, repo.GitPath, defaultBranch, 1, 0)
	if err != nil || len(commits) == 0 {
		h.log.Warn("Failed to get latest commit for CI trigger",
			logger.Error(err),
			logger.String("repo", fmt.Sprintf("%s/%s", owner, repoName)),
			logger.String("branch", defaultBranch),
		)
		return
	}
	latestCommit := commits[0]

	// Check if CI config file exists in the repository
	ciConfigPath := h.ciService.GetConfigPath()
	_, err = h.gitService.GetFileContent(ctx, repo.GitPath, defaultBranch, ciConfigPath)
	if err != nil {
		h.log.Debug("CI config file not found, skipping CI trigger",
			logger.String("repo", fmt.Sprintf("%s/%s", owner, repoName)),
			logger.String("config_path", ciConfigPath),
		)
		return
	}

	// Build the clone URL for CI runner
	cloneURL := h.ciService.BuildCloneURL(owner, repoName)

	// Determine trigger actor
	triggerActor := "anonymous"
	if user != nil {
		triggerActor = user.Username
	}

	// Trigger the CI job asynchronously
	go func() {
		triggerCtx := context.Background()

		h.log.Info("Triggering CI job after push",
			logger.String("repo", fmt.Sprintf("%s/%s", owner, repoName)),
			logger.String("branch", defaultBranch),
			logger.String("commit", latestCommit.Hash),
			logger.String("actor", triggerActor),
			logger.String("clone_url", cloneURL),
			logger.String("config_path", ciConfigPath),
		)

		job, err := h.ciService.TriggerJob(triggerCtx, &service.TriggerJobRequest{
			RepositoryID: repo.ID,
			Owner:        owner,
			RepoName:     repoName,
			CloneURL:     cloneURL,
			CommitSHA:    latestCommit.Hash,
			RefName:      defaultBranch,
			RefType:      models.CIRefTypeBranch,
			TriggerType:  models.CITriggerTypePush,
			TriggerActor: triggerActor,
			Metadata: map[string]string{
				"commit_message": latestCommit.Message,
				"author":         latestCommit.Author,
				"author_email":   latestCommit.AuthorEmail,
			},
		})

		if err != nil {
			h.log.Error("Failed to trigger CI job after push",
				logger.Error(err),
				logger.String("repo", fmt.Sprintf("%s/%s", owner, repoName)),
				logger.String("commit", latestCommit.Hash),
			)
			return
		}

		h.log.Info("CI job triggered successfully",
			logger.String("job_id", job.ID.String()),
			logger.String("repo", fmt.Sprintf("%s/%s", owner, repoName)),
			logger.String("commit", latestCommit.Hash),
		)
	}()
}

// checkRepoAccess checks if the user can access the repository
func (h *GitHandler) checkRepoAccess(c *gin.Context, user *models.User, repo *models.Repository, isWrite bool) bool {
	// Public repos allow read access to everyone
	if !repo.IsPrivate && !isWrite {
		return true
	}

	// Must be authenticated for private repos or write access
	if user == nil {
		c.Header("WWW-Authenticate", `Basic realm="Git Server"`)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return false
	}

	// Check if user has access
	var hasAccess bool
	if user.IsAdmin {
		hasAccess = true
	} else if user.ID == repo.OwnerID {
		hasAccess = true
	} else if !isWrite && !repo.IsPrivate {
		// Read access to public repos
		hasAccess = true
	}
	// TODO: Check collaborator permissions

	if !hasAccess {
		// Return 404 for private repos to avoid leaking existence
		if repo.IsPrivate {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Repository not found",
			})
		} else {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "You don't have permission to access this repository",
			})
		}
		return false
	}

	return true
}

// HandleGetHEAD handles GET /{owner}/{repo}/HEAD (dumb protocol fallback)
func (h *GitHandler) HandleGetHEAD(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	repoName = strings.TrimSuffix(repoName, ".git")

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	user := middleware.GetUserFromContext(c)
	if !h.checkRepoAccess(c, user, repo, false) {
		return
	}

	headPath := fmt.Sprintf("%s/HEAD", repo.GitPath)
	data, err := h.storage.ReadFile(headPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	c.Header("Content-Type", "text/plain")
	c.Writer.Write(data)
}

// HandleGetObject handles GET /{owner}/{repo}/objects/:dir/:file or /objects/pack/:packfile (dumb protocol)
func (h *GitHandler) HandleGetObject(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	repoName = strings.TrimSuffix(repoName, ".git")

	// Build object path from route params
	// Handles both loose objects (/:dir/:file) and pack files (/pack/:packfile)
	var objectPath string
	if packfile := c.Param("packfile"); packfile != "" {
		objectPath = "pack/" + packfile
	} else {
		dir := c.Param("dir")
		file := c.Param("file")
		objectPath = dir + "/" + file
	}

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	user := middleware.GetUserFromContext(c)
	if !h.checkRepoAccess(c, user, repo, false) {
		return
	}

	// Build full path
	fullPath := fmt.Sprintf("%s/objects/%s", repo.GitPath, objectPath)

	// Open and stream file
	reader, err := h.storage.OpenFile(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	defer reader.Close()

	// Determine content type
	contentType := "application/x-git-loose-object"
	if strings.HasSuffix(objectPath, ".pack") {
		contentType = "application/x-git-packed-objects"
	} else if strings.HasSuffix(objectPath, ".idx") {
		contentType = "application/x-git-packed-objects-toc"
	}

	c.Header("Content-Type", contentType)
	io.Copy(c.Writer, reader)
}

// HandleGetRefs handles GET /{owner}/{repo}/refs/heads/:branch or /refs/tags/:tag (dumb protocol)
func (h *GitHandler) HandleGetRefs(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	repoName = strings.TrimSuffix(repoName, ".git")

	// Build ref path from route params
	var refPath string
	if branch := c.Param("branch"); branch != "" {
		refPath = "heads/" + branch
	} else if tag := c.Param("tag"); tag != "" {
		refPath = "tags/" + tag
	}

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	user := middleware.GetUserFromContext(c)
	if !h.checkRepoAccess(c, user, repo, false) {
		return
	}

	fullPath := fmt.Sprintf("%s/refs/%s", repo.GitPath, refPath)

	data, err := h.storage.ReadFile(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	c.Header("Content-Type", "text/plain")
	c.Writer.Write(data)
}

// HandleGetInfoPacks handles GET /{owner}/{repo}/objects/info/packs (dumb protocol)
func (h *GitHandler) HandleGetInfoPacks(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	repoName = strings.TrimSuffix(repoName, ".git")

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	user := middleware.GetUserFromContext(c)
	if !h.checkRepoAccess(c, user, repo, false) {
		return
	}

	packsPath := fmt.Sprintf("%s/objects/info/packs", repo.GitPath)

	data, err := h.storage.ReadFile(packsPath)
	if err != nil {
		// Return empty response if file doesn't exist
		c.Header("Content-Type", "text/plain")
		c.Writer.WriteHeader(http.StatusOK)
		return
	}

	c.Header("Content-Type", "text/plain")
	c.Writer.Write(data)
}
