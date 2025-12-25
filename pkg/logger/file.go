package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// fileWriter is a file writer with log rotation support
// It provides functionality similar to lumberjack but without external dependencies
type fileWriter struct {
	// Filename is the file to write logs to
	Filename string

	// MaxSize is the maximum size in megabytes of the log file before rotation
	MaxSize int

	// MaxBackups is the maximum number of old log files to retain
	MaxBackups int

	// MaxAge is the maximum number of days to retain old log files
	MaxAge int

	// Compress determines if rotated log files should be compressed (gzip)
	Compress bool

	mu       sync.Mutex
	file     *os.File
	size     int64
	rotating bool
}

// Write implements io.Writer
func (w *fileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	writeLen := int64(len(p))

	// Open file if not already open
	if w.file == nil {
		if err := w.openFile(); err != nil {
			return 0, err
		}
	}

	// Check if rotation is needed
	if w.size+writeLen > w.maxBytes() {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = w.file.Write(p)
	w.size += int64(n)

	return n, err
}

// Close implements io.Closer
func (w *fileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.closeFile()
}

// Sync syncs the file to disk
func (w *fileWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Sync()
	}
	return nil
}

// maxBytes returns the maximum file size in bytes
func (w *fileWriter) maxBytes() int64 {
	if w.MaxSize <= 0 {
		return 100 * 1024 * 1024 // 100 MB default
	}
	return int64(w.MaxSize) * 1024 * 1024
}

// openFile opens the log file for writing
func (w *fileWriter) openFile() error {
	// Ensure directory exists
	dir := filepath.Dir(w.Filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open or create the file
	file, err := os.OpenFile(w.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	w.file = file
	w.size = info.Size()

	return nil
}

// closeFile closes the current log file
func (w *fileWriter) closeFile() error {
	if w.file == nil {
		return nil
	}

	err := w.file.Close()
	w.file = nil
	w.size = 0
	return err
}

// rotate performs log rotation
func (w *fileWriter) rotate() error {
	// Close current file
	if err := w.closeFile(); err != nil {
		return err
	}

	// Generate new backup filename
	backupName := w.backupName()

	// Rename current file to backup
	if err := os.Rename(w.Filename, backupName); err != nil {
		// If rename fails, try to continue with a new file
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to rotate log file: %w", err)
		}
	}

	// Open new file
	if err := w.openFile(); err != nil {
		return err
	}

	// Start cleanup in background
	go w.cleanup()

	return nil
}

// backupName generates a backup filename with timestamp
func (w *fileWriter) backupName() string {
	dir := filepath.Dir(w.Filename)
	filename := filepath.Base(w.Filename)
	ext := filepath.Ext(filename)
	name := filename[:len(filename)-len(ext)]

	timestamp := time.Now().Format("2006-01-02T15-04-05.000")

	return filepath.Join(dir, fmt.Sprintf("%s-%s%s", name, timestamp, ext))
}

// cleanup removes old log files based on MaxBackups and MaxAge
func (w *fileWriter) cleanup() {
	w.mu.Lock()
	if w.rotating {
		w.mu.Unlock()
		return
	}
	w.rotating = true
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.rotating = false
		w.mu.Unlock()
	}()

	// List all backup files
	backups, err := w.listBackups()
	if err != nil {
		return
	}

	// Remove old backups by count
	if w.MaxBackups > 0 && len(backups) > w.MaxBackups {
		for _, backup := range backups[w.MaxBackups:] {
			os.Remove(backup.path)
		}
		backups = backups[:w.MaxBackups]
	}

	// Remove old backups by age
	if w.MaxAge > 0 {
		cutoff := time.Now().AddDate(0, 0, -w.MaxAge)
		for _, backup := range backups {
			if backup.modTime.Before(cutoff) {
				os.Remove(backup.path)
			}
		}
	}
}

// backupInfo holds information about a backup file
type backupInfo struct {
	path    string
	modTime time.Time
}

// listBackups returns a list of backup files sorted by modification time (newest first)
func (w *fileWriter) listBackups() ([]backupInfo, error) {
	dir := filepath.Dir(w.Filename)
	filename := filepath.Base(w.Filename)
	ext := filepath.Ext(filename)
	name := filename[:len(filename)-len(ext)]

	pattern := filepath.Join(dir, name+"-*"+ext)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var backups []backupInfo
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		backups = append(backups, backupInfo{
			path:    match,
			modTime: info.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.After(backups[j].modTime)
	})

	return backups, nil
}

// ensureLogDir ensures the directory for the log file exists
func ensureLogDir(filePath string) error {
	dir := filepath.Dir(filePath)
	return os.MkdirAll(dir, 0755)
}

// Ensure fileWriter implements io.WriteCloser
var _ io.WriteCloser = (*fileWriter)(nil)
