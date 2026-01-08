package storage

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bravo68web/stasis/internal/domain/service"
)

// FilesystemStorage implements the StorageService interface for local filesystem
type FilesystemStorage struct {
	basePath string
}

// NewFilesystemStorage creates a new filesystem storage instance
func NewFilesystemStorage(basePath string) (*FilesystemStorage, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return &FilesystemStorage{
		basePath: absPath,
	}, nil
}

// GetRepoPath returns the full path for a repository given owner and repo name
func (s *FilesystemStorage) GetRepoPath(owner, repoName string) string {
	return filepath.Join(s.basePath, owner, repoName+".git")
}

// GetBasePath returns the base storage path
func (s *FilesystemStorage) GetBasePath() string {
	return s.basePath
}

// Exists checks if a path exists in the storage
func (s *FilesystemStorage) Exists(path string) (bool, error) {
	fullPath := s.resolvePath(path)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check path existence: %w", err)
	}
	return true, nil
}

// IsDir checks if the path is a directory
func (s *FilesystemStorage) IsDir(path string) (bool, error) {
	fullPath := s.resolvePath(path)
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat path: %w", err)
	}
	return info.IsDir(), nil
}

// CreateDirectory creates a directory and all parent directories
func (s *FilesystemStorage) CreateDirectory(path string) error {
	fullPath := s.resolvePath(path)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

// DeleteDirectory removes a directory and all its contents
func (s *FilesystemStorage) DeleteDirectory(path string) error {
	fullPath := s.resolvePath(path)
	if err := os.RemoveAll(fullPath); err != nil {
		return fmt.Errorf("failed to delete directory: %w", err)
	}
	return nil
}

// ReadFile reads the entire file content
func (s *FilesystemStorage) ReadFile(path string) ([]byte, error) {
	fullPath := s.resolvePath(path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return data, nil
}

// WriteFile writes data to a file, creating it if it doesn't exist
func (s *FilesystemStorage) WriteFile(path string, data []byte) error {
	fullPath := s.resolvePath(path)

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// OpenFile opens a file for reading
func (s *FilesystemStorage) OpenFile(path string) (io.ReadCloser, error) {
	fullPath := s.resolvePath(path)
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

// CreateFile creates or truncates a file for writing
func (s *FilesystemStorage) CreateFile(path string) (io.WriteCloser, error) {
	fullPath := s.resolvePath(path)

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create parent directory: %w", err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	return file, nil
}

// AppendFile opens a file for appending
func (s *FilesystemStorage) AppendFile(path string) (io.WriteCloser, error) {
	fullPath := s.resolvePath(path)

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create parent directory: %w", err)
	}

	file, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for appending: %w", err)
	}
	return file, nil
}

// DeleteFile removes a file
func (s *FilesystemStorage) DeleteFile(path string) error {
	fullPath := s.resolvePath(path)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, consider it deleted
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// CopyFile copies a file from source to destination
func (s *FilesystemStorage) CopyFile(src, dst string) error {
	srcPath := s.resolvePath(src)
	dstPath := s.resolvePath(dst)

	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Ensure destination directory exists
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Get source file info for permissions
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Apply same permissions
	if err := os.Chmod(dstPath, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// MoveFile moves/renames a file
func (s *FilesystemStorage) MoveFile(src, dst string) error {
	srcPath := s.resolvePath(src)
	dstPath := s.resolvePath(dst)

	// Ensure destination directory exists
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}
	return nil
}

// Stat returns file info for the given path
func (s *FilesystemStorage) Stat(path string) (fs.FileInfo, error) {
	fullPath := s.resolvePath(path)
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path not found: %s", path)
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}
	return info, nil
}

// ListFiles returns a list of file paths in the given directory
func (s *FilesystemStorage) ListFiles(path string) ([]string, error) {
	fullPath := s.resolvePath(path)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// ReadDir reads a directory and returns directory entries
func (s *FilesystemStorage) ReadDir(path string) ([]fs.DirEntry, error) {
	fullPath := s.resolvePath(path)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	return entries, nil
}

// Walk walks the file tree rooted at root, calling fn for each file or directory
func (s *FilesystemStorage) Walk(root string, fn filepath.WalkFunc) error {
	fullPath := s.resolvePath(root)
	return filepath.Walk(fullPath, fn)
}

// CreateSymlink creates a symbolic link
func (s *FilesystemStorage) CreateSymlink(target, link string) error {
	linkPath := s.resolvePath(link)

	// Ensure parent directory exists
	linkDir := filepath.Dir(linkPath)
	if err := os.MkdirAll(linkDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	if err := os.Symlink(target, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}
	return nil
}

// ReadSymlink reads the target of a symbolic link
func (s *FilesystemStorage) ReadSymlink(path string) (string, error) {
	fullPath := s.resolvePath(path)
	target, err := os.Readlink(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read symlink: %w", err)
	}
	return target, nil
}

// Chmod changes the permissions of a file
func (s *FilesystemStorage) Chmod(path string, mode fs.FileMode) error {
	fullPath := s.resolvePath(path)
	if err := os.Chmod(fullPath, mode); err != nil {
		return fmt.Errorf("failed to change file permissions: %w", err)
	}
	return nil
}

// Size returns the size of a file in bytes
func (s *FilesystemStorage) Size(path string) (int64, error) {
	fullPath := s.resolvePath(path)
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("file not found: %s", path)
		}
		return 0, fmt.Errorf("failed to stat file: %w", err)
	}
	return info.Size(), nil
}

// GetDiskUsage returns the total size of a directory in bytes
func (s *FilesystemStorage) GetDiskUsage(path string) (int64, error) {
	fullPath := s.resolvePath(path)
	var size int64

	err := filepath.Walk(fullPath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to calculate disk usage: %w", err)
	}

	return size, nil
}

// resolvePath resolves a relative path to an absolute path within the base directory
func (s *FilesystemStorage) resolvePath(path string) string {
	// If path is already absolute and within base path, return it
	if filepath.IsAbs(path) && strings.HasPrefix(path, s.basePath) {
		return path
	}
	// Otherwise, join with base path
	return filepath.Join(s.basePath, path)
}

// SyncToRemote is a no-op for filesystem storage
// since files are already on the local filesystem
func (s *FilesystemStorage) SyncToRemote(localPath string) error {
	// No-op for filesystem storage
	return nil
}

// Verify interface compliance at compile time
var _ service.StorageService = (*FilesystemStorage)(nil)
