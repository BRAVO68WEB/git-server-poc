package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/bravo68web/githut/internal/domain/service"
)

// S3Storage implements the StorageService interface using AWS S3
type S3Storage struct {
	client     *s3.Client
	bucket     string
	prefix     string // Base prefix for all objects (e.g., "repos/")
	mu         sync.RWMutex
	localCache string // Local cache directory for temporary files
}

// S3Config holds configuration for S3 storage
type S3Config struct {
	Bucket       string
	Region       string
	AccessKey    string
	SecretKey    string
	Endpoint     string // Optional: for S3-compatible services like MinIO
	UsePathStyle bool   // Optional: use path-style addressing
	Prefix       string // Base prefix for all objects
	LocalCache   string // Local cache directory
}

// NewS3Storage creates a new S3 storage instance
func NewS3Storage(ctx context.Context, cfg S3Config) (*S3Storage, error) {
	// Build AWS config options
	var configOpts []func(*config.LoadOptions) error
	configOpts = append(configOpts, config.WithRegion(cfg.Region))

	// Add credentials if provided
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		configOpts = append(configOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}

	// Load AWS configuration
	awsCfg, err := config.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with optional custom endpoint
	var s3Opts []func(*s3.Options)
	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = cfg.UsePathStyle
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Opts...)

	// Ensure prefix ends with /
	prefix := cfg.Prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// Set up local cache directory
	localCache := cfg.LocalCache
	if localCache == "" {
		localCache = os.TempDir()
	}

	storage := &S3Storage{
		client:     client,
		bucket:     cfg.Bucket,
		prefix:     prefix,
		localCache: localCache,
	}

	// Verify bucket exists
	if err := storage.verifyBucket(ctx); err != nil {
		return nil, fmt.Errorf("failed to verify S3 bucket: %w", err)
	}

	return storage, nil
}

// verifyBucket checks if the bucket exists and is accessible
func (s *S3Storage) verifyBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	return err
}

// GetRepoPath returns the S3 key path for a repository
func (s *S3Storage) GetRepoPath(owner, repoName string) string {
	return fmt.Sprintf("%s%s/%s.git", s.prefix, owner, repoName)
}

// GetBasePath returns the base prefix
func (s *S3Storage) GetBasePath() string {
	return s.prefix
}

// fullKey returns the full S3 key for a path
func (s *S3Storage) fullKey(path string) string {
	// If path already has prefix, don't add it again
	if strings.HasPrefix(path, s.prefix) {
		return path
	}
	return s.prefix + path
}

// Exists checks if an object or prefix exists in S3
func (s *S3Storage) Exists(path string) (bool, error) {
	ctx := context.Background()
	key := s.fullKey(path)

	// First try as object
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return true, nil
	}

	// Check if it's a "directory" (prefix)
	result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucket),
		Prefix:  aws.String(key + "/"),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return len(result.Contents) > 0, nil
}

// IsDir checks if the path is a "directory" (prefix with objects beneath it)
func (s *S3Storage) IsDir(path string) (bool, error) {
	ctx := context.Background()
	key := s.fullKey(path)

	if !strings.HasSuffix(key, "/") {
		key += "/"
	}

	result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucket),
		Prefix:  aws.String(key),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return false, fmt.Errorf("failed to check if directory: %w", err)
	}

	return len(result.Contents) > 0 || len(result.CommonPrefixes) > 0, nil
}

// CreateDirectory creates a "directory" in S3 (a zero-byte object with trailing /)
func (s *S3Storage) CreateDirectory(path string) error {
	ctx := context.Background()
	key := s.fullKey(path)

	if !strings.HasSuffix(key, "/") {
		key += "/"
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader([]byte{}),
	})
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

// DeleteDirectory removes all objects with the given prefix
func (s *S3Storage) DeleteDirectory(path string) error {
	ctx := context.Background()
	key := s.fullKey(path)

	if !strings.HasSuffix(key, "/") {
		key += "/"
	}

	// List all objects with prefix
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(key),
	})

	var objectsToDelete []types.ObjectIdentifier

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects for deletion: %w", err)
		}

		for _, obj := range page.Contents {
			objectsToDelete = append(objectsToDelete, types.ObjectIdentifier{
				Key: obj.Key,
			})
		}
	}

	if len(objectsToDelete) == 0 {
		return nil // Nothing to delete
	}

	// Delete in batches of 1000 (S3 limit)
	for i := 0; i < len(objectsToDelete); i += 1000 {
		end := i + 1000
		if end > len(objectsToDelete) {
			end = len(objectsToDelete)
		}

		_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &types.Delete{
				Objects: objectsToDelete[i:end],
				Quiet:   aws.Bool(true),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to delete objects: %w", err)
		}
	}

	return nil
}

// ReadFile reads the entire content of an object
func (s *S3Storage) ReadFile(path string) ([]byte, error) {
	ctx := context.Background()
	key := s.fullKey(path)

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	defer result.Body.Close()

	return io.ReadAll(result.Body)
}

// WriteFile writes data to an object
func (s *S3Storage) WriteFile(path string, data []byte) error {
	ctx := context.Background()
	key := s.fullKey(path)

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// s3ReadCloser wraps an S3 GetObject response body
type s3ReadCloser struct {
	body io.ReadCloser
}

func (r *s3ReadCloser) Read(p []byte) (n int, err error) {
	return r.body.Read(p)
}

func (r *s3ReadCloser) Close() error {
	return r.body.Close()
}

// OpenFile opens an object for reading
func (s *S3Storage) OpenFile(path string) (io.ReadCloser, error) {
	ctx := context.Background()
	key := s.fullKey(path)

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return &s3ReadCloser{body: result.Body}, nil
}

// s3WriteCloser implements io.WriteCloser for S3 uploads
type s3WriteCloser struct {
	storage *S3Storage
	key     string
	buf     *bytes.Buffer
	closed  bool
	mu      sync.Mutex
}

func (w *s3WriteCloser) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, errors.New("write to closed file")
	}
	return w.buf.Write(p)
}

func (w *s3WriteCloser) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}
	w.closed = true

	ctx := context.Background()
	_, err := w.storage.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(w.storage.bucket),
		Key:    aws.String(w.key),
		Body:   bytes.NewReader(w.buf.Bytes()),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

// CreateFile creates a new object for writing
func (s *S3Storage) CreateFile(path string) (io.WriteCloser, error) {
	key := s.fullKey(path)
	return &s3WriteCloser{
		storage: s,
		key:     key,
		buf:     bytes.NewBuffer(nil),
	}, nil
}

// AppendFile opens an object for appending (reads existing content first)
func (s *S3Storage) AppendFile(path string) (io.WriteCloser, error) {
	key := s.fullKey(path)

	// Read existing content
	existingData, err := s.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to read existing file for append: %w", err)
	}

	buf := bytes.NewBuffer(existingData)
	return &s3WriteCloser{
		storage: s,
		key:     key,
		buf:     buf,
	}, nil
}

// DeleteFile removes an object
func (s *S3Storage) DeleteFile(path string) error {
	ctx := context.Background()
	key := s.fullKey(path)

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// CopyFile copies an object from source to destination
func (s *S3Storage) CopyFile(src, dst string) error {
	ctx := context.Background()
	srcKey := s.fullKey(src)
	dstKey := s.fullKey(dst)

	copySource := fmt.Sprintf("%s/%s", s.bucket, srcKey)

	_, err := s.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(dstKey),
	})
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// MoveFile moves/renames an object
func (s *S3Storage) MoveFile(src, dst string) error {
	if err := s.CopyFile(src, dst); err != nil {
		return err
	}
	return s.DeleteFile(src)
}

// s3FileInfo implements fs.FileInfo for S3 objects
type s3FileInfo struct {
	name    string
	size    int64
	modTime time.Time
	isDir   bool
}

func (f *s3FileInfo) Name() string { return f.name }
func (f *s3FileInfo) Size() int64  { return f.size }
func (f *s3FileInfo) Mode() fs.FileMode {
	if f.isDir {
		return fs.ModeDir | 0755
	}
	return 0644
}
func (f *s3FileInfo) ModTime() time.Time { return f.modTime }
func (f *s3FileInfo) IsDir() bool        { return f.isDir }
func (f *s3FileInfo) Sys() interface{}   { return nil }

// Stat returns file info for an object
func (s *S3Storage) Stat(path string) (fs.FileInfo, error) {
	ctx := context.Background()
	key := s.fullKey(path)

	// Try as object first
	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return &s3FileInfo{
			name:    filepath.Base(path),
			size:    aws.ToInt64(result.ContentLength),
			modTime: aws.ToTime(result.LastModified),
			isDir:   false,
		}, nil
	}

	// Try as directory
	isDir, err := s.IsDir(path)
	if err != nil {
		return nil, err
	}
	if isDir {
		return &s3FileInfo{
			name:    filepath.Base(path),
			size:    0,
			modTime: time.Now(),
			isDir:   true,
		}, nil
	}

	return nil, os.ErrNotExist
}

// ListFiles returns a list of file paths in a "directory"
func (s *S3Storage) ListFiles(path string) ([]string, error) {
	ctx := context.Background()
	key := s.fullKey(path)

	if !strings.HasSuffix(key, "/") {
		key += "/"
	}

	var files []string

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(key),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}

		for _, obj := range page.Contents {
			// Remove the prefix and base key to get relative path
			relPath := strings.TrimPrefix(aws.ToString(obj.Key), key)
			if relPath != "" {
				files = append(files, relPath)
			}
		}
	}

	sort.Strings(files)
	return files, nil
}

// s3DirEntry implements fs.DirEntry for S3 objects
type s3DirEntry struct {
	info *s3FileInfo
}

func (e *s3DirEntry) Name() string               { return e.info.name }
func (e *s3DirEntry) IsDir() bool                { return e.info.isDir }
func (e *s3DirEntry) Type() fs.FileMode          { return e.info.Mode().Type() }
func (e *s3DirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

// ReadDir reads a "directory" and returns directory entries
func (s *S3Storage) ReadDir(path string) ([]fs.DirEntry, error) {
	ctx := context.Background()
	key := s.fullKey(path)

	if !strings.HasSuffix(key, "/") {
		key += "/"
	}

	result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.bucket),
		Prefix:    aws.String(key),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var entries []fs.DirEntry

	// Add directories (common prefixes)
	for _, prefix := range result.CommonPrefixes {
		name := strings.TrimPrefix(aws.ToString(prefix.Prefix), key)
		name = strings.TrimSuffix(name, "/")
		if name != "" {
			entries = append(entries, &s3DirEntry{
				info: &s3FileInfo{
					name:    name,
					size:    0,
					modTime: time.Now(),
					isDir:   true,
				},
			})
		}
	}

	// Add files
	for _, obj := range result.Contents {
		name := strings.TrimPrefix(aws.ToString(obj.Key), key)
		if name != "" && !strings.Contains(name, "/") {
			entries = append(entries, &s3DirEntry{
				info: &s3FileInfo{
					name:    name,
					size:    aws.ToInt64(obj.Size),
					modTime: aws.ToTime(obj.LastModified),
					isDir:   false,
				},
			})
		}
	}

	// Sort entries by name
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	return entries, nil
}

// Walk walks the file tree rooted at root, calling fn for each file or directory
func (s *S3Storage) Walk(root string, fn filepath.WalkFunc) error {
	ctx := context.Background()
	key := s.fullKey(root)

	if !strings.HasSuffix(key, "/") {
		key += "/"
	}

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(key),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to walk: %w", err)
		}

		for _, obj := range page.Contents {
			objKey := aws.ToString(obj.Key)
			relPath := strings.TrimPrefix(objKey, s.prefix)

			info := &s3FileInfo{
				name:    filepath.Base(relPath),
				size:    aws.ToInt64(obj.Size),
				modTime: aws.ToTime(obj.LastModified),
				isDir:   strings.HasSuffix(objKey, "/"),
			}

			if err := fn(relPath, info, nil); err != nil {
				if errors.Is(err, filepath.SkipDir) {
					continue
				}
				return err
			}
		}
	}

	return nil
}

// CreateSymlink is not supported in S3 (symlinks don't exist in object storage)
func (s *S3Storage) CreateSymlink(target, link string) error {
	return errors.New("symlinks are not supported in S3 storage")
}

// ReadSymlink is not supported in S3
func (s *S3Storage) ReadSymlink(path string) (string, error) {
	return "", errors.New("symlinks are not supported in S3 storage")
}

// Chmod is not supported in S3
func (s *S3Storage) Chmod(path string, mode fs.FileMode) error {
	// S3 doesn't have Unix permissions, so this is a no-op
	return nil
}

// Size returns the size of an object
func (s *S3Storage) Size(path string) (int64, error) {
	info, err := s.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetDiskUsage returns the total size of all objects under a prefix
func (s *S3Storage) GetDiskUsage(path string) (int64, error) {
	ctx := context.Background()
	key := s.fullKey(path)

	if !strings.HasSuffix(key, "/") {
		key += "/"
	}

	var totalSize int64

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(key),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to calculate disk usage: %w", err)
		}

		for _, obj := range page.Contents {
			totalSize += aws.ToInt64(obj.Size)
		}
	}

	return totalSize, nil
}

// Verify interface compliance at compile time
var _ service.StorageService = (*S3Storage)(nil)
