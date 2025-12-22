package handler

import (
	"net/http"
	"strconv"

	"github.com/bravo68web/stasis/internal/application/dto"
	"github.com/bravo68web/stasis/internal/application/service"
	"github.com/bravo68web/stasis/internal/transport/http/middleware"
	apperrors "github.com/bravo68web/stasis/pkg/errors"
	"github.com/gin-gonic/gin"
)

// RepoHandler handles repository-related HTTP requests
type RepoHandler struct {
	repoService *service.RepoService
	baseURL     string
	sshHost     string
	sshPort     int
}

// NewRepoHandler creates a new RepoHandler instance
func NewRepoHandler(
	repoService *service.RepoService,
	baseURL string,
	sshHost string,
	sshPort int,
) *RepoHandler {
	return &RepoHandler{
		repoService: repoService,
		baseURL:     baseURL,
		sshHost:     sshHost,
		sshPort:     sshPort,
	}
}

// CreateRepository handles POST /api/repos
func (h *RepoHandler) CreateRepository(c *gin.Context) {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	var req dto.CreateRepoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": err.Error(),
		})
		return
	}

	// Create repository
	repo, err := h.repoService.CreateRepository(
		c.Request.Context(),
		user.ID,
		req.Name,
		req.Description,
		req.IsPrivate,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Build response
	response := dto.RepoFromModel(repo, h.baseURL, h.sshHost, h.sshPort)

	c.JSON(http.StatusCreated, response)
}

// ListRepositories handles GET /api/repos
func (h *RepoHandler) ListRepositories(c *gin.Context) {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	repos, err := h.repoService.ListUserRepositories(c.Request.Context(), user.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Convert to response DTOs
	responses := make([]dto.RepoResponse, len(repos))
	for i, repo := range repos {
		responses[i] = dto.RepoFromModel(repo, h.baseURL, h.sshHost, h.sshPort)
	}

	c.JSON(http.StatusOK, gin.H{
		"repositories": responses,
		"total":        len(responses),
	})
}

// ListPublicRepositories handles GET /api/repos/public
func (h *RepoHandler) ListPublicRepositories(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	repos, err := h.repoService.ListPublicRepositories(c.Request.Context(), perPage, offset)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Convert to response DTOs
	responses := make([]dto.RepoResponse, len(repos))
	for i, repo := range repos {
		responses[i] = dto.RepoFromModel(repo, h.baseURL, h.sshHost, h.sshPort)
	}

	c.JSON(http.StatusOK, gin.H{
		"repositories": responses,
		"page":         page,
		"per_page":     perPage,
		"total":        len(responses),
	})
}

// GetRepository handles GET /api/repos/:owner/:repo
func (h *RepoHandler) GetRepository(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check access for private repos
	user := middleware.GetUserFromContext(c)
	if repo.IsPrivate {
		if user == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Repository not found",
			})
			return
		}
		if user.ID != repo.OwnerID && !user.IsAdmin {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Repository not found",
			})
			return
		}
	}

	response := dto.RepoFromModel(repo, h.baseURL, h.sshHost, h.sshPort)
	c.JSON(http.StatusOK, response)
}

// UpdateRepository handles PATCH /api/repos/:owner/:repo
func (h *RepoHandler) UpdateRepository(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check ownership
	if user.ID != repo.OwnerID && !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "You don't have permission to update this repository",
		})
		return
	}

	var req dto.UpdateRepoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid request body",
		})
		return
	}

	updatedRepo, err := h.repoService.UpdateRepository(
		c.Request.Context(),
		repo.ID,
		req.Description,
		req.IsPrivate,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response := dto.RepoFromModel(updatedRepo, h.baseURL, h.sshHost, h.sshPort)
	c.JSON(http.StatusOK, response)
}

// DeleteRepository handles DELETE /api/repos/:owner/:repo
func (h *RepoHandler) DeleteRepository(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check ownership
	if user.ID != repo.OwnerID && !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "You don't have permission to delete this repository",
		})
		return
	}

	if err := h.repoService.DeleteRepository(c.Request.Context(), repo.ID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Repository deleted successfully",
	})
}

// ListBranches handles GET /api/repos/:owner/:repo/branches
func (h *RepoHandler) ListBranches(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check access
	user := middleware.GetUserFromContext(c)
	if repo.IsPrivate && (user == nil || (user.ID != repo.OwnerID && !user.IsAdmin)) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Repository not found",
		})
		return
	}

	branches, err := h.repoService.ListBranches(c.Request.Context(), repo)
	if err != nil {
		h.handleError(c, err)
		return
	}

	responses := make([]dto.BranchResponse, len(branches))
	for i, branch := range branches {
		responses[i] = dto.BranchResponse{
			Name:   branch.Name,
			Hash:   branch.Hash,
			IsHead: branch.IsHead,
		}
	}

	c.JSON(http.StatusOK, dto.BranchListResponse{
		Branches: responses,
		Total:    len(responses),
	})
}

// CreateBranch handles POST /api/repos/:owner/:repo/branches
func (h *RepoHandler) CreateBranch(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check write access
	if user.ID != repo.OwnerID && !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "You don't have permission to create branches",
		})
		return
	}

	var req dto.BranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid request body",
		})
		return
	}

	if err := h.repoService.CreateBranch(c.Request.Context(), repo, req.Name, req.CommitHash); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Branch created successfully",
		"branch":  req.Name,
	})
}

// DeleteBranch handles DELETE /api/repos/:owner/:repo/branches/:branch
func (h *RepoHandler) DeleteBranch(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	branchName := c.Param("branch")

	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check write access
	if user.ID != repo.OwnerID && !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "You don't have permission to delete branches",
		})
		return
	}

	if err := h.repoService.DeleteBranch(c.Request.Context(), repo, branchName); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Branch deleted successfully",
	})
}

// ListTags handles GET /api/repos/:owner/:repo/tags
func (h *RepoHandler) ListTags(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check access
	user := middleware.GetUserFromContext(c)
	if repo.IsPrivate && (user == nil || (user.ID != repo.OwnerID && !user.IsAdmin)) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Repository not found",
		})
		return
	}

	tags, err := h.repoService.ListTags(c.Request.Context(), repo)
	if err != nil {
		h.handleError(c, err)
		return
	}

	responses := make([]dto.TagResponse, len(tags))
	for i, tag := range tags {
		responses[i] = dto.TagResponse{
			Name:        tag.Name,
			Hash:        tag.Hash,
			Message:     tag.Message,
			Tagger:      tag.Tagger,
			IsAnnotated: !tag.IsLight,
		}
	}

	c.JSON(http.StatusOK, dto.TagListResponse{
		Tags:  responses,
		Total: len(responses),
	})
}

// CreateTag handles POST /api/repos/:owner/:repo/tags
func (h *RepoHandler) CreateTag(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check write access
	if user.ID != repo.OwnerID && !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "You don't have permission to create tags",
		})
		return
	}

	var req dto.TagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid request body",
		})
		return
	}

	if err := h.repoService.CreateTag(c.Request.Context(), repo, req.Name, req.CommitHash, req.Message); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Tag created successfully",
		"tag":     req.Name,
	})
}

// DeleteTag handles DELETE /api/repos/:owner/:repo/tags/:tag
func (h *RepoHandler) DeleteTag(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	tagName := c.Param("tag")

	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check write access
	if user.ID != repo.OwnerID && !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "You don't have permission to delete tags",
		})
		return
	}

	if err := h.repoService.DeleteTag(c.Request.Context(), repo, tagName); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tag deleted successfully",
	})
}

// GetRepositoryStats handles GET /api/repos/:owner/:repo/stats
func (h *RepoHandler) GetRepositoryStats(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check access
	user := middleware.GetUserFromContext(c)
	if repo.IsPrivate && (user == nil || (user.ID != repo.OwnerID && !user.IsAdmin)) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Repository not found",
		})
		return
	}

	stats, err := h.repoService.GetRepositoryStats(c.Request.Context(), repo)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ListCommits handles GET /api/repos/:owner/:repo/commits
func (h *RepoHandler) ListCommits(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check access
	user := middleware.GetUserFromContext(c)
	if repo.IsPrivate && (user == nil || (user.ID != repo.OwnerID && !user.IsAdmin)) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Repository not found",
		})
		return
	}

	// Get query parameters
	ref := c.DefaultQuery("ref", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "30"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	offset := (page - 1) * perPage

	commits, err := h.repoService.GetCommits(c.Request.Context(), repo, ref, perPage, offset)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response := dto.CommitListFromService(commits, ref)
	response.Ref = ref
	if ref == "" {
		response.Ref = "HEAD"
	}

	c.JSON(http.StatusOK, response)
}

// GetCommit handles GET /api/repos/:owner/:repo/commits/:sha
func (h *RepoHandler) GetCommit(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	sha := c.Param("sha")

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check access
	user := middleware.GetUserFromContext(c)
	if repo.IsPrivate && (user == nil || (user.ID != repo.OwnerID && !user.IsAdmin)) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Repository not found",
		})
		return
	}

	commit, err := h.repoService.GetCommit(c.Request.Context(), repo, sha)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Commit not found",
		})
		return
	}

	response := dto.CommitFromService(*commit)
	c.JSON(http.StatusOK, response)
}

// GetDiff handles GET /api/v1/repos/:owner/:repo/diff/:hash
// Returns the patch content for a commit.
func (h *RepoHandler) GetDiff(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	hash := c.Param("hash")

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Access control: allow public reads; require auth for private repos
	user := middleware.GetUserFromContext(c)
	if repo.IsPrivate && (user == nil || (user.ID != repo.OwnerID && !user.IsAdmin)) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Repository not found",
		})
		return
	}

	if hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Commit hash is required",
		})
		return
	}

	diffResult, err := h.repoService.GetDiff(c.Request.Context(), repo, hash)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Diff not found",
			"details": err.Error(),
		})
		return
	}

	response := dto.DiffFromService(diffResult)
	c.JSON(http.StatusOK, response)
}

// GetTree handles GET /api/repos/:owner/:repo/tree/:ref/*path
func (h *RepoHandler) GetTree(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Param("ref")
	path := c.Param("path")

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check access
	user := middleware.GetUserFromContext(c)
	if repo.IsPrivate && (user == nil || (user.ID != repo.OwnerID && !user.IsAdmin)) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Repository not found",
		})
		return
	}

	entries, err := h.repoService.GetTree(c.Request.Context(), repo, ref, path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Tree not found",
			"details": err.Error(),
		})
		return
	}

	response := dto.TreeFromService(entries, path, ref)
	c.JSON(http.StatusOK, response)
}

// GetFileContent handles GET /api/repos/:owner/:repo/blob/:ref/*path
func (h *RepoHandler) GetFileContent(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Param("ref")
	path := c.Param("path")

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check access
	user := middleware.GetUserFromContext(c)
	if repo.IsPrivate && (user == nil || (user.ID != repo.OwnerID && !user.IsAdmin)) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Repository not found",
		})
		return
	}

	fileContent, err := h.repoService.GetFileContent(c.Request.Context(), repo, ref, path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "File not found",
			"details": err.Error(),
		})
		return
	}

	response := dto.FileContentFromService(fileContent, ref)
	c.JSON(http.StatusOK, response)
}

// GetBlame handles GET /api/repos/:owner/:repo/blame/:ref/*path
func (h *RepoHandler) GetBlame(c *gin.Context) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Param("ref")
	path := c.Param("path")

	repo, err := h.repoService.GetRepository(c.Request.Context(), owner, repoName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Check access
	user := middleware.GetUserFromContext(c)
	if repo.IsPrivate && (user == nil || (user.ID != repo.OwnerID && !user.IsAdmin)) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Repository not found",
		})
		return
	}

	blameLines, err := h.repoService.GetBlame(c.Request.Context(), repo, ref, path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Unable to get blame information",
			"details": err.Error(),
		})
		return
	}

	response := dto.BlameFromService(blameLines, path, ref)
	c.JSON(http.StatusOK, response)
}

// handleError handles errors and returns appropriate HTTP responses
func (h *RepoHandler) handleError(c *gin.Context, err error) {
	if apperrors.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Repository not found",
		})
		return
	}

	if apperrors.IsConflict(err) {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "conflict",
			"message": err.Error(),
		})
		return
	}

	if apperrors.IsForbidden(err) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "internal_error",
		"message": "An internal error occurred",
	})
}
