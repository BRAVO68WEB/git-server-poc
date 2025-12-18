package service

import (
	"context"
	"io"
	"time"
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

// Commit represents a Git commit
type Commit struct {
	Hash           string
	ShortHash      string
	Message        string
	Author         string
	AuthorEmail    string
	AuthorDate     time.Time
	Committer      string
	CommitterEmail string
	CommitterDate  time.Time
	ParentHashes   []string
}

// TreeEntry represents an entry in a Git tree (file or directory)
type TreeEntry struct {
	Name string // File or directory name
	Path string // Full path from repository root
	Type string // "blob" for file, "tree" for directory
	Mode string // File mode (e.g., "100644" for regular file, "040000" for directory)
	Hash string // Object hash
	Size int64  // File size in bytes (only for blobs)
}

// FileContent represents the content of a file in a Git repository
type FileContent struct {
	Path     string
	Name     string
	Size     int64
	Hash     string
	Content  []byte
	IsBinary bool
	Encoding string // "utf-8", "base64" for binary files
}

// BlameLine represents a single line in a blame output
type BlameLine struct {
	LineNo  int
	Commit  string
	Author  string
	Email   string
	Date    time.Time
	Content string
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

	// Commit operations
	// GetCommits returns a list of commits for a given ref (branch/tag/commit hash)
	// If ref is empty, uses the default branch
	GetCommits(ctx context.Context, repoPath, ref string, limit, offset int) ([]Commit, error)

	// GetCommit returns a single commit by hash
	GetCommit(ctx context.Context, repoPath, commitHash string) (*Commit, error)

	// Tree operations
	// GetTree returns the tree entries for a given ref and path
	// If path is empty, returns the root tree
	GetTree(ctx context.Context, repoPath, ref, path string) ([]TreeEntry, error)

	// File operations
	// GetFileContent returns the content of a file at a given ref and path
	GetFileContent(ctx context.Context, repoPath, ref, filePath string) (*FileContent, error)

	// Blame operations
	// GetBlame returns blame information for a file at a given ref
	GetBlame(ctx context.Context, repoPath, ref, filePath string) ([]BlameLine, error)
}
