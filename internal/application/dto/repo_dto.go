package dto

import (
	"encoding/base64"
	"time"

	"github.com/bravo68web/githut/internal/domain/models"
	"github.com/bravo68web/githut/internal/domain/service"
	"github.com/google/uuid"
)

// CreateRepoRequest represents a request to create a new repository
type CreateRepoRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	IsPrivate   bool   `json:"is_private"`
}

// UpdateRepoRequest represents a request to update a repository
type UpdateRepoRequest struct {
	Description *string `json:"description,omitempty"`
	IsPrivate   *bool   `json:"is_private,omitempty"`
}

// RepoResponse represents the response for repository data
type RepoResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Owner       string    `json:"owner"`
	OwnerID     uuid.UUID `json:"owner_id"`
	IsPrivate   bool      `json:"is_private"`
	Description string    `json:"description"`
	CloneURL    string    `json:"clone_url"`
	SSHURL      string    `json:"ssh_url"`
	GitPath     string    `json:"git_path,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RepoListResponse represents a paginated list of repositories
type RepoListResponse struct {
	Repositories []RepoResponse `json:"repositories"`
	Total        int64          `json:"total"`
	Page         int            `json:"page"`
	PerPage      int            `json:"per_page"`
	TotalPages   int            `json:"total_pages"`
}

// BranchRequest represents a request to create a new branch
type BranchRequest struct {
	Name       string `json:"name" binding:"required,min=1,max=255"`
	CommitHash string `json:"commit_hash" binding:"required,len=40"`
}

// BranchResponse represents the response for branch data
type BranchResponse struct {
	Name   string `json:"name"`
	Hash   string `json:"hash"`
	IsHead bool   `json:"is_head"`
}

// BranchListResponse represents a list of branches
type BranchListResponse struct {
	Branches []BranchResponse `json:"branches"`
	Total    int              `json:"total"`
}

// TagRequest represents a request to create a new tag
type TagRequest struct {
	Name       string `json:"name" binding:"required,min=1,max=255"`
	CommitHash string `json:"commit_hash" binding:"required,len=40"`
	Message    string `json:"message"` // Optional: if provided, creates annotated tag
}

// TagResponse represents the response for tag data
type TagResponse struct {
	Name        string `json:"name"`
	Hash        string `json:"hash"`
	Message     string `json:"message,omitempty"`
	Tagger      string `json:"tagger,omitempty"`
	IsAnnotated bool   `json:"is_annotated"`
}

// TagListResponse represents a list of tags
type TagListResponse struct {
	Tags  []TagResponse `json:"tags"`
	Total int           `json:"total"`
}

// CommitResponse represents a commit in API responses
type CommitResponse struct {
	Hash           string    `json:"hash"`
	ShortHash      string    `json:"short_hash"`
	Message        string    `json:"message"`
	Author         string    `json:"author"`
	AuthorEmail    string    `json:"author_email"`
	AuthorDate     time.Time `json:"author_date"`
	Committer      string    `json:"committer"`
	CommitterEmail string    `json:"committer_email"`
	CommitterDate  time.Time `json:"committer_date"`
	ParentHashes   []string  `json:"parent_hashes"`
}

// CommitListResponse represents a list of commits
type CommitListResponse struct {
	Commits []CommitResponse `json:"commits"`
	Total   int              `json:"total"`
	Ref     string           `json:"ref"`
}

// TreeEntryResponse represents a tree entry (file or directory) in API responses
type TreeEntryResponse struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // "blob" for file, "tree" for directory
	Mode string `json:"mode"`
	Hash string `json:"hash"`
	Size int64  `json:"size,omitempty"` // Only for blobs
}

// TreeResponse represents a tree listing in API responses
type TreeResponse struct {
	Entries []TreeEntryResponse `json:"entries"`
	Path    string              `json:"path"`
	Ref     string              `json:"ref"`
	Total   int                 `json:"total"`
}

// FileContentResponse represents file content in API responses
type FileContentResponse struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Hash     string `json:"hash"`
	Content  string `json:"content"`
	IsBinary bool   `json:"is_binary"`
	Encoding string `json:"encoding"` // "utf-8" or "base64"
	Ref      string `json:"ref"`
}

// CommitFromService converts a service.Commit to CommitResponse DTO
func CommitFromService(c service.Commit) CommitResponse {
	return CommitResponse{
		Hash:           c.Hash,
		ShortHash:      c.ShortHash,
		Message:        c.Message,
		Author:         c.Author,
		AuthorEmail:    c.AuthorEmail,
		AuthorDate:     c.AuthorDate,
		Committer:      c.Committer,
		CommitterEmail: c.CommitterEmail,
		CommitterDate:  c.CommitterDate,
		ParentHashes:   c.ParentHashes,
	}
}

// CommitListFromService converts a slice of service.Commit to CommitListResponse
func CommitListFromService(commits []service.Commit, ref string) CommitListResponse {
	responses := make([]CommitResponse, len(commits))
	for i, c := range commits {
		responses[i] = CommitFromService(c)
	}
	return CommitListResponse{
		Commits: responses,
		Total:   len(responses),
		Ref:     ref,
	}
}

// TreeEntryFromService converts a service.TreeEntry to TreeEntryResponse DTO
func TreeEntryFromService(e service.TreeEntry) TreeEntryResponse {
	return TreeEntryResponse{
		Name: e.Name,
		Path: e.Path,
		Type: e.Type,
		Mode: e.Mode,
		Hash: e.Hash,
		Size: e.Size,
	}
}

// TreeFromService converts a slice of service.TreeEntry to TreeResponse
func TreeFromService(entries []service.TreeEntry, path, ref string) TreeResponse {
	responses := make([]TreeEntryResponse, len(entries))
	for i, e := range entries {
		responses[i] = TreeEntryFromService(e)
	}
	return TreeResponse{
		Entries: responses,
		Path:    path,
		Ref:     ref,
		Total:   len(responses),
	}
}

// FileContentFromService converts a service.FileContent to FileContentResponse DTO
func FileContentFromService(f *service.FileContent, ref string) FileContentResponse {
	content := string(f.Content)
	if f.IsBinary {
		content = base64.StdEncoding.EncodeToString(f.Content)
	}

	return FileContentResponse{
		Path:     f.Path,
		Name:     f.Name,
		Size:     f.Size,
		Hash:     f.Hash,
		Content:  content,
		IsBinary: f.IsBinary,
		Encoding: f.Encoding,
		Ref:      ref,
	}
}

// RepoFromModel converts a Repository model to RepoResponse DTO
func RepoFromModel(repo *models.Repository, baseURL, sshHost string, sshPort int) RepoResponse {
	response := RepoResponse{
		ID:          repo.ID,
		Name:        repo.Name,
		OwnerID:     repo.OwnerID,
		IsPrivate:   repo.IsPrivate,
		Description: repo.Description,
		GitPath:     repo.GitPath,
		CreatedAt:   repo.CreatedAt,
		UpdatedAt:   repo.UpdatedAt,
	}

	// Set owner username if available
	if repo.Owner.Username != "" {
		response.Owner = repo.Owner.Username
	}

	// Generate clone URLs
	if baseURL != "" && response.Owner != "" {
		response.CloneURL = buildCloneURL(baseURL, response.Owner, repo.Name)
	}
	if sshHost != "" && response.Owner != "" {
		response.SSHURL = buildSSHURL(sshHost, sshPort, response.Owner, repo.Name)
	}

	return response
}

// RepoListFromModels converts a slice of Repository models to RepoListResponse
func RepoListFromModels(repos []*models.Repository, total int64, page, perPage int, baseURL, sshHost string, sshPort int) RepoListResponse {
	responses := make([]RepoResponse, len(repos))
	for i, repo := range repos {
		responses[i] = RepoFromModel(repo, baseURL, sshHost, sshPort)
	}

	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	return RepoListResponse{
		Repositories: responses,
		Total:        total,
		Page:         page,
		PerPage:      perPage,
		TotalPages:   totalPages,
	}
}

// buildCloneURL builds the HTTP clone URL for a repository
func buildCloneURL(baseURL, owner, name string) string {
	return baseURL + "/" + owner + "/" + name + ".git"
}

// buildSSHURL builds the SSH clone URL for a repository
func buildSSHURL(host string, port int, owner, name string) string {
	if port == 22 {
		return "git@" + host + ":" + owner + "/" + name + ".git"
	}
	return "ssh://git@" + host + ":" + string(rune(port)) + "/" + owner + "/" + name + ".git"
}

// Validate validates the CreateRepoRequest
func (r *CreateRepoRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > 100 {
		return ErrNameTooLong
	}
	if !isValidRepoName(r.Name) {
		return ErrInvalidRepoName
	}
	return nil
}

// isValidRepoName checks if a repository name is valid
func isValidRepoName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	for _, c := range name {
		if !isValidRepoNameChar(c) {
			return false
		}
	}
	return true
}

// isValidRepoNameChar checks if a character is valid in a repository name
func isValidRepoNameChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_' || c == '.'
}

// Common validation errors
var (
	ErrNameRequired    = &ValidationError{Field: "name", Message: "name is required"}
	ErrNameTooLong     = &ValidationError{Field: "name", Message: "name must be 100 characters or less"}
	ErrInvalidRepoName = &ValidationError{Field: "name", Message: "name contains invalid characters"}
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return e.Message
}
