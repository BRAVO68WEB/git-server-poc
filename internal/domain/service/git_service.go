package service

import (
	"context"
	"io"
)

// Ref represents a Git reference (branch or tag)
type Ref struct {
	Name   string
	Hash   string
	Target string // For symbolic refs like HEAD
}

// Tag represents a Git tag
type Tag struct {
	Name    string
	Hash    string
	Message string // For annotated tags
	Tagger  string
	IsLight bool // True if it's a lightweight tag
}

// Branch represents a Git branch
type Branch struct {
	Name   string
	Hash   string
	IsHead bool // True if this is the current HEAD
}

// GitService defines the interface for Git repository operations
type GitService interface {
	// Repository operations
	// InitRepository initializes a new Git repository at the specified path
	// If bare is true, creates a bare repository (no working directory)
	InitRepository(ctx context.Context, repoPath string, bare bool) error

	// CloneRepository clones a repository from source to destination
	CloneRepository(ctx context.Context, source, dest string, bare bool) error

	// DeleteRepository removes a repository from the storage
	DeleteRepository(ctx context.Context, repoPath string) error

	// RepositoryExists checks if a repository exists at the given path
	RepositoryExists(ctx context.Context, repoPath string) (bool, error)

	// Reference operations
	// GetRefs returns all references (branches and tags) in the repository
	GetRefs(ctx context.Context, repoPath string) (map[string]string, error)

	// GetHEAD returns the current HEAD reference
	GetHEAD(ctx context.Context, repoPath string) (string, error)

	// Branch operations
	// CreateBranch creates a new branch pointing to the specified commit
	CreateBranch(ctx context.Context, repoPath, branchName, commitHash string) error

	// DeleteBranch removes a branch from the repository
	DeleteBranch(ctx context.Context, repoPath, branchName string) error

	// ListBranches returns all branches in the repository
	ListBranches(ctx context.Context, repoPath string) ([]Branch, error)

	// GetBranch returns information about a specific branch
	GetBranch(ctx context.Context, repoPath, branchName string) (*Branch, error)

	// Tag operations
	// CreateTag creates a new tag. If message is empty, creates a lightweight tag
	CreateTag(ctx context.Context, repoPath, tagName, commitHash, message string) error

	// DeleteTag removes a tag from the repository
	DeleteTag(ctx context.Context, repoPath, tagName string) error

	// ListTags returns all tags in the repository
	ListTags(ctx context.Context, repoPath string) ([]Tag, error)

	// GetTag returns information about a specific tag
	GetTag(ctx context.Context, repoPath, tagName string) (*Tag, error)

	// Protocol operations (Smart HTTP)
	// ReceivePack handles git push operations (git-receive-pack)
	// Reads pack data from input and writes response to output
	ReceivePack(ctx context.Context, repoPath string, input io.Reader, output io.Writer) error

	// UploadPack handles git fetch/pull operations (git-upload-pack)
	// Reads client wants from input and writes pack data to output
	UploadPack(ctx context.Context, repoPath string, input io.Reader, output io.Writer) error

	// GetInfoRefs returns the info/refs content for smart HTTP protocol
	// service should be "git-upload-pack" or "git-receive-pack"
	GetInfoRefs(ctx context.Context, repoPath, service string) ([]byte, error)

	// UpdateServerInfo updates auxiliary info file (for dumb HTTP protocol)
	UpdateServerInfo(ctx context.Context, repoPath string) error

	// Object operations
	// GetObject returns the content of a Git object
	GetObject(ctx context.Context, repoPath, objectHash string) ([]byte, error)

	// ObjectExists checks if an object exists in the repository
	ObjectExists(ctx context.Context, repoPath, objectHash string) (bool, error)

	// GetDefaultBranch returns the default branch name for the repository
	GetDefaultBranch(ctx context.Context, repoPath string) (string, error)
}
