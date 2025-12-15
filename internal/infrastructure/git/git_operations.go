package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bravo68web/githut/internal/domain/service"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GitOperations implements the GitService interface using go-git library
type GitOperations struct {
	storage service.StorageService
}

// NewGitOperations creates a new GitOperations instance
func NewGitOperations(storage service.StorageService) *GitOperations {
	return &GitOperations{
		storage: storage,
	}
}

// InitRepository initializes a new Git repository at the specified path
func (g *GitOperations) InitRepository(ctx context.Context, repoPath string, bare bool) error {
	// Ensure the directory exists
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	// Initialize the repository
	_, err := git.PlainInit(repoPath, bare)
	if err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	// For bare repos, update server info to support dumb HTTP protocol
	if bare {
		if err := g.UpdateServerInfo(ctx, repoPath); err != nil {
			// TODO: Log the error but don't fail the init
		}
	}

	return nil
}

// CloneRepository clones a repository from source to destination
func (g *GitOperations) CloneRepository(ctx context.Context, source, dest string, bare bool) error {
	_, err := git.PlainClone(dest, bare, &git.CloneOptions{
		URL:      source,
		Progress: nil,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// DeleteRepository removes a repository from the storage
func (g *GitOperations) DeleteRepository(ctx context.Context, repoPath string) error {
	if err := os.RemoveAll(repoPath); err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}

	return nil
}

// RepositoryExists checks if a repository exists at the given path
func (g *GitOperations) RepositoryExists(ctx context.Context, repoPath string) (bool, error) {
	_, err := git.PlainOpen(repoPath)
	if err != nil {
		if err == git.ErrRepositoryNotExists {
			return false, nil
		}
		return false, fmt.Errorf("failed to check repository: %w", err)
	}
	return true, nil
}

// GetRefs returns all references (branches and tags) in the repository
func (g *GitOperations) GetRefs(ctx context.Context, repoPath string) (map[string]string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	refs := make(map[string]string)

	// Get all references
	refIter, err := repo.References()
	if err != nil {
		return nil, fmt.Errorf("failed to get references: %w", err)
	}

	err = refIter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() == plumbing.HashReference {
			refs[ref.Name().String()] = ref.Hash().String()
		} else if ref.Type() == plumbing.SymbolicReference {
			refs[ref.Name().String()] = ref.Target().String()
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate references: %w", err)
	}

	return refs, nil
}

// GetHEAD returns the current HEAD reference
func (g *GitOperations) GetHEAD(ctx context.Context, repoPath string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return "", nil
		}
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	return head.Hash().String(), nil
}

// CreateBranch creates a new branch pointing to the specified commit
func (g *GitOperations) CreateBranch(ctx context.Context, repoPath, branchName, commitHash string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Validate commit hash
	hash := plumbing.NewHash(commitHash)
	_, err = repo.CommitObject(hash)
	if err != nil {
		return fmt.Errorf("invalid commit hash: %w", err)
	}

	// Create branch reference
	refName := plumbing.NewBranchReferenceName(branchName)
	ref := plumbing.NewHashReference(refName, hash)

	err = repo.Storer.SetReference(ref)
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	return nil
}

// DeleteBranch removes a branch from the repository
func (g *GitOperations) DeleteBranch(ctx context.Context, repoPath, branchName string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Check if it's the HEAD branch
	head, err := repo.Head()
	if err == nil && head.Name().Short() == branchName {
		return fmt.Errorf("cannot delete the current HEAD branch")
	}

	refName := plumbing.NewBranchReferenceName(branchName)
	err = repo.Storer.RemoveReference(refName)
	if err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	return nil
}

// ListBranches returns all branches in the repository
func (g *GitOperations) ListBranches(ctx context.Context, repoPath string) ([]service.Branch, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get HEAD for comparison
	head, _ := repo.Head()
	headName := ""
	if head != nil {
		headName = head.Name().Short()
	}

	branches := []service.Branch{}

	refIter, err := repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	err = refIter.ForEach(func(ref *plumbing.Reference) error {
		branches = append(branches, service.Branch{
			Name:   ref.Name().Short(),
			Hash:   ref.Hash().String(),
			IsHead: ref.Name().Short() == headName,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate branches: %w", err)
	}

	return branches, nil
}

// GetBranch returns information about a specific branch
func (g *GitOperations) GetBranch(ctx context.Context, repoPath, branchName string) (*service.Branch, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	refName := plumbing.NewBranchReferenceName(branchName)
	ref, err := repo.Reference(refName, true)
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}

	head, _ := repo.Head()
	isHead := head != nil && head.Name().Short() == branchName

	return &service.Branch{
		Name:   branchName,
		Hash:   ref.Hash().String(),
		IsHead: isHead,
	}, nil
}

// CreateTag creates a new tag. If message is empty, creates a lightweight tag
func (g *GitOperations) CreateTag(ctx context.Context, repoPath, tagName, commitHash, message string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	hash := plumbing.NewHash(commitHash)

	if message == "" {
		// Lightweight tag
		refName := plumbing.NewTagReferenceName(tagName)
		ref := plumbing.NewHashReference(refName, hash)
		err = repo.Storer.SetReference(ref)
		if err != nil {
			return fmt.Errorf("failed to create lightweight tag: %w", err)
		}
	} else {
		// Annotated tag
		_, err = repo.CreateTag(tagName, hash, &git.CreateTagOptions{
			Message: message,
			Tagger: &object.Signature{
				Name:  "Git Server",
				Email: "git@server.local",
				When:  time.Now(),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create annotated tag: %w", err)
		}
	}

	return nil
}

// DeleteTag removes a tag from the repository
func (g *GitOperations) DeleteTag(ctx context.Context, repoPath, tagName string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	err = repo.DeleteTag(tagName)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	return nil
}

// ListTags returns all tags in the repository
func (g *GitOperations) ListTags(ctx context.Context, repoPath string) ([]service.Tag, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	tags := []service.Tag{}

	tagRefs, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	err = tagRefs.ForEach(func(ref *plumbing.Reference) error {
		tag := service.Tag{
			Name:    ref.Name().Short(),
			Hash:    ref.Hash().String(),
			IsLight: true,
		}

		// Try to get annotated tag info
		tagObj, err := repo.TagObject(ref.Hash())
		if err == nil {
			tag.Message = tagObj.Message
			tag.Tagger = tagObj.Tagger.String()
			tag.IsLight = false
		}

		tags = append(tags, tag)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate tags: %w", err)
	}

	return tags, nil
}

// GetTag returns information about a specific tag
func (g *GitOperations) GetTag(ctx context.Context, repoPath, tagName string) (*service.Tag, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	refName := plumbing.NewTagReferenceName(tagName)
	ref, err := repo.Reference(refName, true)
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	tag := &service.Tag{
		Name:    tagName,
		Hash:    ref.Hash().String(),
		IsLight: true,
	}

	// Try to get annotated tag info
	tagObj, err := repo.TagObject(ref.Hash())
	if err == nil {
		tag.Message = tagObj.Message
		tag.Tagger = tagObj.Tagger.String()
		tag.IsLight = false
	}

	return tag, nil
}

// ReceivePack handles git push operations (git-receive-pack)
func (g *GitOperations) ReceivePack(ctx context.Context, repoPath string, input io.Reader, output io.Writer) error {
	// Use git command for receive-pack as go-git doesn't fully support server-side receive-pack
	cmd := exec.CommandContext(ctx, "git", "receive-pack", "--stateless-rpc", repoPath)
	cmd.Stdin = input
	cmd.Stdout = output
	cmd.Stderr = os.Stderr
	cmd.Dir = repoPath

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("receive-pack failed: %w", err)
	}

	// Update server info after receiving push
	if err := g.UpdateServerInfo(ctx, repoPath); err != nil {
		// TODO: Log the error but don't fail the receive-pack
	}

	return nil
}

// UploadPack handles git fetch/pull operations (git-upload-pack)
func (g *GitOperations) UploadPack(ctx context.Context, repoPath string, input io.Reader, output io.Writer) error {
	// Use git command for upload-pack as it handles pack negotiation
	cmd := exec.CommandContext(ctx, "git", "upload-pack", "--stateless-rpc", repoPath)
	cmd.Stdin = input
	cmd.Stdout = output
	cmd.Stderr = os.Stderr
	cmd.Dir = repoPath

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("upload-pack failed: %w", err)
	}

	return nil
}

// GetInfoRefs returns the info/refs content for smart HTTP protocol
func (g *GitOperations) GetInfoRefs(ctx context.Context, repoPath, serviceName string) ([]byte, error) {
	var buf bytes.Buffer

	// Build service advertisement header
	service := serviceName
	if !strings.HasPrefix(service, "git-") {
		service = "git-" + service
	}

	// Write pkt-line header
	header := fmt.Sprintf("# service=%s\n", service)
	pktHeader := fmt.Sprintf("%04x%s", len(header)+4, header)
	buf.WriteString(pktHeader)
	buf.WriteString("0000") // Flush packet

	// Get refs using git command
	cmd := exec.CommandContext(ctx, "git", serviceName, "--stateless-rpc", "--advertise-refs", repoPath)
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	cmd.Dir = repoPath

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to advertise refs: %w", err)
	}

	return buf.Bytes(), nil
}

// UpdateServerInfo updates auxiliary info file (for dumb HTTP protocol)
func (g *GitOperations) UpdateServerInfo(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "update-server-info")
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update server info: %w", err)
	}

	return nil
}

// GetObject returns the content of a Git object
func (g *GitOperations) GetObject(ctx context.Context, repoPath, objectHash string) ([]byte, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	hash := plumbing.NewHash(objectHash)
	obj, err := repo.Storer.EncodedObject(plumbing.AnyObject, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	reader, err := obj.Reader()
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// ObjectExists checks if an object exists in the repository
func (g *GitOperations) ObjectExists(ctx context.Context, repoPath, objectHash string) (bool, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return false, fmt.Errorf("failed to open repository: %w", err)
	}

	hash := plumbing.NewHash(objectHash)
	_, err = repo.Storer.EncodedObject(plumbing.AnyObject, hash)
	if err != nil {
		if err == plumbing.ErrObjectNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object: %w", err)
	}

	return true, nil
}

// SetConfig sets a configuration value in the repository
func (g *GitOperations) SetConfig(ctx context.Context, repoPath, section, key, value string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	cfg, err := repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	cfg.Raw.SetOption(section, "", key, value)

	err = repo.SetConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	return nil
}

// GetConfig gets a configuration value from the repository
func (g *GitOperations) GetConfig(ctx context.Context, repoPath, section, key string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	cfg, err := repo.Config()
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}

	return cfg.Raw.Section(section).Option(key), nil
}

// AddRemote adds a remote to the repository
func (g *GitOperations) AddRemote(ctx context.Context, repoPath, name, url string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: name,
		URLs: []string{url},
	})
	if err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	return nil
}

// RemoveRemote removes a remote from the repository
func (g *GitOperations) RemoveRemote(ctx context.Context, repoPath, name string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	err = repo.DeleteRemote(name)
	if err != nil {
		return fmt.Errorf("failed to remove remote: %w", err)
	}

	return nil
}

// GetDefaultBranch returns the default branch name (usually main or master)
func (g *GitOperations) GetDefaultBranch(ctx context.Context, repoPath string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	// Check HEAD reference
	head, err := repo.Head()
	if err == nil {
		return head.Name().Short(), nil
	}

	// If HEAD doesn't exist, try to find main or master
	branches := []string{"main", "master"}
	for _, branch := range branches {
		refName := plumbing.NewBranchReferenceName(branch)
		_, err := repo.Reference(refName, true)
		if err == nil {
			return branch, nil
		}
	}

	return "", nil
}

// GetObjectPath returns the file path for a loose object
func GetObjectPath(repoPath, objectHash string) string {
	if len(objectHash) < 3 {
		return ""
	}
	return filepath.Join(repoPath, "objects", objectHash[:2], objectHash[2:])
}

// GetPackPath returns the pack directory path
func GetPackPath(repoPath string) string {
	return filepath.Join(repoPath, "objects", "pack")
}

// Verify interface compliance at compile time
var _ service.GitService = (*GitOperations)(nil)
