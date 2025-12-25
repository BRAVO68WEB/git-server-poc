package service

import (
	"io"
	"io/fs"
	"path/filepath"
)

// StorageService defines the interface for storage operations
// This abstraction allows for different storage backends (filesystem, S3, etc.)
type StorageService interface {
	// Path operations

	// GetRepoPath returns the full path for a repository given owner and repo name
	GetRepoPath(owner, repoName string) string

	// GetBasePath returns the base storage path
	GetBasePath() string

	// File existence operations

	// Exists checks if a path exists in the storage
	Exists(path string) (bool, error)

	// IsDir checks if the path is a directory
	IsDir(path string) (bool, error)

	// Directory operations

	// CreateDirectory creates a directory and all parent directories
	CreateDirectory(path string) error

	// DeleteDirectory removes a directory and all its contents
	DeleteDirectory(path string) error

	// File operations

	// ReadFile reads the entire file content
	ReadFile(path string) ([]byte, error)

	// WriteFile writes data to a file, creating it if it doesn't exist
	WriteFile(path string, data []byte) error

	// OpenFile opens a file for reading
	OpenFile(path string) (io.ReadCloser, error)

	// CreateFile creates or truncates a file for writing
	CreateFile(path string) (io.WriteCloser, error)

	// AppendFile opens a file for appending
	AppendFile(path string) (io.WriteCloser, error)

	// DeleteFile removes a file
	DeleteFile(path string) error

	// CopyFile copies a file from source to destination
	CopyFile(src, dst string) error

	// MoveFile moves/renames a file
	MoveFile(src, dst string) error

	// Stat returns file info for the given path
	Stat(path string) (fs.FileInfo, error)

	// Listing operations

	// ListFiles returns a list of file paths in the given directory
	ListFiles(path string) ([]string, error)

	// ReadDir reads a directory and returns directory entries
	ReadDir(path string) ([]fs.DirEntry, error)

	// Walk walks the file tree rooted at root, calling fn for each file or directory
	Walk(root string, fn filepath.WalkFunc) error

	// Symlink operations (useful for git)

	// CreateSymlink creates a symbolic link
	CreateSymlink(target, link string) error

	// ReadSymlink reads the target of a symbolic link
	ReadSymlink(path string) (string, error)

	// Permission operations

	// Chmod changes the permissions of a file
	Chmod(path string, mode fs.FileMode) error

	// Size operations

	// Size returns the size of a file in bytes
	Size(path string) (int64, error)

	// GetDiskUsage returns the total size of a directory in bytes
	GetDiskUsage(path string) (int64, error)
}
