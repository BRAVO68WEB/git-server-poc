package dto

import (
	"encoding/base64"
	"time"

	"github.com/bravo68web/stasis/internal/domain/models"
	"github.com/bravo68web/stasis/internal/domain/service"
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
	Description   *string `json:"description,omitempty"`
	IsPrivate     *bool   `json:"is_private,omitempty"`
	DefaultBranch *string `json:"default_branch,omitempty"`
}

// ImportRepoRequest represents a request to import a repository from an external Git source
type ImportRepoRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	IsPrivate   bool   `json:"is_private"`
	CloneURL    string `json:"clone_url" binding:"required,url"`
	Username    string `json:"username,omitempty"` // Optional: for authentication
	Password    string `json:"password,omitempty"` // Optional: for authentication (can be personal access token)
	Mirror      bool   `json:"mirror"`             // If true, creates a mirror repository
}

// RepoResponse represents the response for repository data
type RepoResponse struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Owner           string     `json:"owner"`
	OwnerID         uuid.UUID  `json:"owner_id"`
	IsPrivate       bool       `json:"is_private"`
	Description     string     `json:"description"`
	DefaultBranch   string     `json:"default_branch"`
	CloneURL        string     `json:"clone_url"`
	SSHURL          string     `json:"ssh_url"`
	GitPath         string     `json:"git_path,omitempty"`
	MirrorEnabled   bool       `json:"mirror_enabled"`
	MirrorDirection string     `json:"mirror_direction,omitempty"`
	UpstreamURL     string     `json:"upstream_url,omitempty"`
	DownstreamURL   string     `json:"downstream_url,omitempty"`
	SyncInterval    int        `json:"sync_interval"`
	SyncSchedule    string     `json:"sync_schedule,omitempty"`
	LastSyncedAt    *time.Time `json:"last_synced_at,omitempty"`
	NextSyncAt      *time.Time `json:"next_sync_at,omitempty"`
	SyncStatus      string     `json:"sync_status,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
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

// BlameLineResponse represents a single line in blame output for API responses
type BlameLineResponse struct {
	LineNo  int    `json:"line_no"`
	Commit  string `json:"commit"`
	Author  string `json:"author"`
	Email   string `json:"email,omitempty"`
	Date    string `json:"date"`
	Content string `json:"content"`
}

// BlameResponse represents blame information for a file in API responses
type BlameResponse struct {
	Blame []BlameLineResponse `json:"blame"`
	Path  string              `json:"path"`
	Ref   string              `json:"ref"`
	Total int                 `json:"total"`
}

// BlameLineFromService converts a service.BlameLine to BlameLineResponse DTO
func BlameLineFromService(b service.BlameLine) BlameLineResponse {
	return BlameLineResponse{
		LineNo:  b.LineNo,
		Commit:  b.Commit,
		Author:  b.Author,
		Email:   b.Email,
		Date:    b.Date.Format(time.RFC3339),
		Content: b.Content,
	}
}

// BlameFromService converts a slice of service.BlameLine to BlameResponse
func BlameFromService(lines []service.BlameLine, path, ref string) BlameResponse {
	responses := make([]BlameLineResponse, len(lines))
	for i, line := range lines {
		responses[i] = BlameLineFromService(line)
	}
	return BlameResponse{
		Blame: responses,
		Path:  path,
		Ref:   ref,
		Total: len(responses),
	}
}

// DiffResponse represents the diff output for a commit in API responses
type DiffResponse struct {
	CommitHash   string         `json:"commit_hash"`
	Content      string         `json:"content"`
	FilesChanged int            `json:"files_changed"`
	Additions    int            `json:"additions"`
	Deletions    int            `json:"deletions"`
	Files        []DiffFileInfo `json:"files,omitempty"`
}

// DiffFileInfo represents a single file's diff information in API responses
type DiffFileInfo struct {
	OldPath   string `json:"old_path"`
	NewPath   string `json:"new_path"`
	Status    string `json:"status"` // "added", "deleted", "modified", "renamed"
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch,omitempty"`
}

// DiffFromService converts a service.DiffResult to DiffResponse DTO
func DiffFromService(d *service.DiffResult) DiffResponse {
	var files []DiffFileInfo
	for _, f := range d.Files {
		files = append(files, DiffFileInfo{
			OldPath:   f.OldPath,
			NewPath:   f.NewPath,
			Status:    f.Status,
			Additions: f.Additions,
			Deletions: f.Deletions,
			Patch:     f.Patch,
		})
	}

	return DiffResponse{
		CommitHash:   d.CommitHash,
		Content:      d.Content,
		FilesChanged: d.FilesChanged,
		Additions:    d.Additions,
		Deletions:    d.Deletions,
		Files:        files,
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
		ID:              repo.ID,
		Name:            repo.Name,
		OwnerID:         repo.OwnerID,
		IsPrivate:       repo.IsPrivate,
		Description:     repo.Description,
		DefaultBranch:   repo.DefaultBranch,
		GitPath:         repo.GitPath,
		MirrorEnabled:   repo.MirrorEnabled,
		MirrorDirection: repo.MirrorDirection,
		UpstreamURL:     repo.UpstreamURL,
		DownstreamURL:   repo.DownstreamURL,
		SyncInterval:    repo.SyncInterval,
		SyncSchedule:    repo.SyncSchedule,
		LastSyncedAt:    repo.LastSyncedAt,
		NextSyncAt:      repo.GetNextSyncTime(),
		SyncStatus:      repo.SyncStatus,
		CreatedAt:       repo.CreatedAt,
		UpdatedAt:       repo.UpdatedAt,
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

// RepoStatsResponse represents repository statistics
type RepoStatsResponse struct {
	BranchCount int   `json:"branch_count"`
	TagCount    int   `json:"tag_count"`
	DiskUsage   int64 `json:"disk_usage"`
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

// Validate validates the ImportRepoRequest
func (r *ImportRepoRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > 100 {
		return ErrNameTooLong
	}
	if !isValidRepoName(r.Name) {
		return ErrInvalidRepoName
	}
	if r.CloneURL == "" {
		return ErrCloneURLRequired
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

// UpdateMirrorSettingsRequest represents a request to update mirror settings
type UpdateMirrorSettingsRequest struct {
	MirrorEnabled      *bool   `json:"mirror_enabled,omitempty"`
	MirrorDirection    *string `json:"mirror_direction,omitempty"` // "upstream", "downstream", "both"
	UpstreamURL        *string `json:"upstream_url,omitempty"`
	UpstreamUsername   *string `json:"upstream_username,omitempty"`
	UpstreamPassword   *string `json:"upstream_password,omitempty"`
	DownstreamURL      *string `json:"downstream_url,omitempty"`
	DownstreamUsername *string `json:"downstream_username,omitempty"`
	DownstreamPassword *string `json:"downstream_password,omitempty"`
	SyncInterval       *int    `json:"sync_interval,omitempty"` // in seconds (deprecated, use SyncSchedule)
	SyncSchedule       *string `json:"sync_schedule,omitempty"` // cron expression (e.g., "0 */1 * * *")
}

// MirrorSettingsResponse represents the mirror settings of a repository
type MirrorSettingsResponse struct {
	MirrorEnabled      bool       `json:"mirror_enabled"`
	MirrorDirection    string     `json:"mirror_direction,omitempty"`
	UpstreamURL        string     `json:"upstream_url,omitempty"`
	UpstreamUsername   string     `json:"upstream_username,omitempty"`
	DownstreamURL      string     `json:"downstream_url,omitempty"`
	DownstreamUsername string     `json:"downstream_username,omitempty"`
	SyncInterval       int        `json:"sync_interval"`
	SyncSchedule       string     `json:"sync_schedule,omitempty"`
	LastSyncedAt       *time.Time `json:"last_synced_at,omitempty"`
	NextSyncAt         *time.Time `json:"next_sync_at,omitempty"`
	SyncStatus         string     `json:"sync_status,omitempty"`
	SyncError          string     `json:"sync_error,omitempty"`
}

// Common validation errors
var (
	ErrNameRequired     = &ValidationError{Field: "name", Message: "name is required"}
	ErrNameTooLong      = &ValidationError{Field: "name", Message: "name must be 100 characters or less"}
	ErrInvalidRepoName  = &ValidationError{Field: "name", Message: "name contains invalid characters"}
	ErrCloneURLRequired = &ValidationError{Field: "clone_url", Message: "clone URL is required"}
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
