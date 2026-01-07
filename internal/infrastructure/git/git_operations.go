package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bravo68web/stasis/internal/domain/service"
	"github.com/bravo68web/stasis/pkg/logger"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GitOperations implements the GitService interface using go-git library
type GitOperations struct {
	storage service.StorageService
	log     *logger.Logger
}

// NewGitOperations creates a new GitOperations instance
func NewGitOperations(storage service.StorageService) service.GitService {
	return &GitOperations{
		storage: storage,
		log:     logger.Get().WithFields(logger.Component("git-operations")),
	}
}

// InitRepository initializes a new Git repository at the specified path
func (g *GitOperations) InitRepository(ctx context.Context, repoPath string, bare bool) error {
	g.log.Info("Initializing git repository",
		logger.String("repo_path", repoPath),
		logger.Bool("bare", bare),
	)

	// Ensure the directory exists
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		g.log.Error("Failed to create repository directory",
			logger.Error(err),
			logger.String("repo_path", repoPath),
		)
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	// Initialize the repository
	_, err := git.PlainInit(repoPath, bare)
	if err != nil {
		g.log.Error("Failed to initialize git repository",
			logger.Error(err),
			logger.String("repo_path", repoPath),
		)
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	// For bare repos, update server info to support dumb HTTP protocol
	if bare {
		if err := g.UpdateServerInfo(ctx, repoPath); err != nil {
			g.log.Warn("Failed to update server info after init",
				logger.Error(err),
				logger.String("repo_path", repoPath),
			)
		}
	}

	g.log.Info("Git repository initialized successfully",
		logger.String("repo_path", repoPath),
	)

	return nil
}

// CloneRepository clones a repository from source to destination
func (g *GitOperations) CloneRepository(ctx context.Context, source, dest string, bare bool) error {
	g.log.Info("Cloning git repository",
		logger.String("source", source),
		logger.String("dest", dest),
		logger.Bool("bare", bare),
	)

	_, err := git.PlainClone(dest, bare, &git.CloneOptions{
		URL:      source,
		Progress: nil,
	})
	if err != nil {
		g.log.Error("Failed to clone repository",
			logger.Error(err),
			logger.String("source", source),
			logger.String("dest", dest),
		)
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	g.log.Info("Repository cloned successfully",
		logger.String("source", source),
		logger.String("dest", dest),
	)

	return nil
}

// DeleteRepository removes a repository from the storage
func (g *GitOperations) DeleteRepository(ctx context.Context, repoPath string) error {
	g.log.Info("Deleting git repository",
		logger.String("repo_path", repoPath),
	)

	if err := os.RemoveAll(repoPath); err != nil {
		g.log.Error("Failed to delete repository",
			logger.Error(err),
			logger.String("repo_path", repoPath),
		)
		return fmt.Errorf("failed to delete repository: %w", err)
	}

	g.log.Info("Repository deleted successfully",
		logger.String("repo_path", repoPath),
	)

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

// GetHEADRef returns the current HEAD reference
func (g *GitOperations) GetHEADRef(ctx context.Context, repoPath string) (string, error) {
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
			Name:        ref.Name().Short(),
			Hash:        ref.Hash().String(),
			IsHead:      ref.Name().Short() == headName,
			CommitCount: g.CountCommits(ctx, repoPath, ref.Name().Short()),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate branches: %w", err)
	}

	return branches, nil
}

func (g *GitOperations) CountCommits(ctx context.Context, repoPath string, branchName string) int {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return 0
	}

	commits, err := repo.CommitObjects()
	if err != nil {
		return 0
	}

	var commitCount int
	err = commits.ForEach(func(commit *object.Commit) error {
		commitCount++
		return nil
	})
	if err != nil {
		return 0
	}

	return int(commitCount)
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

// GetHEADBranch returns the default branch name (usually main or master)
func (g *GitOperations) GetHEADBranch(ctx context.Context, repoPath string) (string, error) {
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

// SetHEADBranch sets the default branch (HEAD) for a bare repository
func (g *GitOperations) SetHEADBranch(ctx context.Context, repoPath, branchName string) error {
	g.log.Debug("Setting default branch",
		logger.String("repo_path", repoPath),
		logger.String("branch", branchName),
	)

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Verify the branch exists
	refName := plumbing.NewBranchReferenceName(branchName)
	_, err = repo.Reference(refName, true)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return fmt.Errorf("branch '%s' does not exist", branchName)
		}
		return fmt.Errorf("failed to verify branch: %w", err)
	}

	// Create a symbolic reference for HEAD pointing to the branch
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, refName)
	err = repo.Storer.SetReference(headRef)
	if err != nil {
		return fmt.Errorf("failed to set HEAD: %w", err)
	}

	g.log.Info("Default branch set successfully",
		logger.String("repo_path", repoPath),
		logger.String("branch", branchName),
	)

	return nil
}

// BranchExists checks if a branch exists in the repository
func (g *GitOperations) BranchExists(ctx context.Context, repoPath, branchName string) (bool, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return false, fmt.Errorf("failed to open repository: %w", err)
	}

	refName := plumbing.NewBranchReferenceName(branchName)
	_, err = repo.Reference(refName, true)
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check branch: %w", err)
	}

	return true, nil
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

// GetCommits returns a list of commits for a given ref
func (g *GitOperations) GetCommits(ctx context.Context, repoPath, ref string, limit, offset int) ([]service.Commit, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Resolve the ref to a commit hash
	hash, err := g.resolveRef(repo, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve ref '%s': %w", ref, err)
	}

	// Get the commit iterator starting from the resolved hash
	commitIter, err := repo.Log(&git.LogOptions{
		From:  hash,
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit log: %w", err)
	}
	defer commitIter.Close()

	commits := []service.Commit{}
	count := 0
	skipped := 0

	err = commitIter.ForEach(func(c *object.Commit) error {
		// Skip commits until we reach the offset
		if skipped < offset {
			skipped++
			return nil
		}

		// Stop if we've reached the limit
		if limit > 0 && count >= limit {
			return fmt.Errorf("limit reached")
		}

		parentHashes := make([]string, len(c.ParentHashes))
		for i, ph := range c.ParentHashes {
			parentHashes[i] = ph.String()
		}

		commits = append(commits, service.Commit{
			Hash:           c.Hash.String(),
			ShortHash:      c.Hash.String()[:7],
			Message:        c.Message,
			Author:         c.Author.Name,
			AuthorEmail:    c.Author.Email,
			AuthorDate:     c.Author.When,
			Committer:      c.Committer.Name,
			CommitterEmail: c.Committer.Email,
			CommitterDate:  c.Committer.When,
			ParentHashes:   parentHashes,
		})

		count++
		return nil
	})

	// Ignore "limit reached" error
	if err != nil && err.Error() != "limit reached" {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return commits, nil
}

// GetCommit returns a single commit by hash
func (g *GitOperations) GetCommit(ctx context.Context, repoPath, commitHash string) (*service.Commit, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	hash := plumbing.NewHash(commitHash)
	c, err := repo.CommitObject(hash)
	if err != nil {
		if err == plumbing.ErrObjectNotFound {
			return nil, fmt.Errorf("commit not found: %s", commitHash)
		}
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	parentHashes := make([]string, len(c.ParentHashes))
	for i, ph := range c.ParentHashes {
		parentHashes[i] = ph.String()
	}

	return &service.Commit{
		Hash:           c.Hash.String(),
		ShortHash:      c.Hash.String()[:7],
		Message:        c.Message,
		Author:         c.Author.Name,
		AuthorEmail:    c.Author.Email,
		AuthorDate:     c.Author.When,
		Committer:      c.Committer.Name,
		CommitterEmail: c.Committer.Email,
		CommitterDate:  c.Committer.When,
		ParentHashes:   parentHashes,
	}, nil
}

// GetTree returns the tree entries for a given ref and path
func (g *GitOperations) GetTree(ctx context.Context, repoPath, ref, path string) ([]service.TreeEntry, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Resolve the ref to a commit hash
	hash, err := g.resolveRef(repo, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve ref '%s': %w", ref, err)
	}

	// Get the commit
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	// Get the tree
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	// If path is specified, navigate to that subtree
	if path != "" && path != "/" {
		path = strings.TrimPrefix(path, "/")
		path = strings.TrimSuffix(path, "/")
		tree, err = tree.Tree(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get subtree at path '%s': %w", path, err)
		}
	}

	entries := []service.TreeEntry{}
	for _, entry := range tree.Entries {
		entryPath := entry.Name
		if path != "" && path != "/" {
			entryPath = path + "/" + entry.Name
		}

		entryType := "blob"
		if entry.Mode == filemode.Dir {
			entryType = "tree"
		} else if entry.Mode == filemode.Submodule {
			entryType = "commit"
		}

		var size int64 = 0
		if entryType == "blob" {
			// Get the blob to retrieve size
			blob, err := repo.BlobObject(entry.Hash)
			if err == nil {
				size = blob.Size
			}
		}

		entries = append(entries, service.TreeEntry{
			Name: entry.Name,
			Path: entryPath,
			Type: entryType,
			Mode: entry.Mode.String(),
			Hash: entry.Hash.String(),
			Size: size,
		})
	}

	return entries, nil
}

// GetFileContent returns the content of a file at a given ref and path
func (g *GitOperations) GetFileContent(ctx context.Context, repoPath, ref, filePath string) (*service.FileContent, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Resolve the ref to a commit hash
	hash, err := g.resolveRef(repo, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve ref '%s': %w", ref, err)
	}

	// Get the commit
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	// Get the file from the tree
	filePath = strings.TrimPrefix(filePath, "/")
	file, err := commit.File(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file '%s': %w", filePath, err)
	}

	// Read the content
	reader, err := file.Reader()
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	// Check if the file is binary
	isBinary := isBinaryContent(content)
	encoding := "utf-8"
	if isBinary {
		encoding = "base64"
	}

	// Extract the file name from the path
	name := filepath.Base(filePath)

	return &service.FileContent{
		Path:     filePath,
		Name:     name,
		Size:     file.Size,
		Hash:     file.Hash.String(),
		Content:  content,
		IsBinary: isBinary,
		Encoding: encoding,
	}, nil
}

// resolveRef resolves a ref string to a commit hash
// It handles branch names, tag names, and commit hashes
func (g *GitOperations) resolveRef(repo *git.Repository, ref string) (plumbing.Hash, error) {
	// If ref is empty, use HEAD
	if ref == "" {
		head, err := repo.Head()
		if err != nil {
			return plumbing.ZeroHash, fmt.Errorf("failed to get HEAD: %w", err)
		}
		return head.Hash(), nil
	}

	// Try as a commit hash first (if it looks like one)
	if len(ref) >= 7 && len(ref) <= 40 {
		hash := plumbing.NewHash(ref)
		if _, err := repo.CommitObject(hash); err == nil {
			return hash, nil
		}
	}

	// Try as a branch name
	branchRef := plumbing.NewBranchReferenceName(ref)
	if r, err := repo.Reference(branchRef, true); err == nil {
		return r.Hash(), nil
	}

	// Try as a tag name
	tagRef := plumbing.NewTagReferenceName(ref)
	if r, err := repo.Reference(tagRef, true); err == nil {
		// For annotated tags, we need to get the target commit
		tagObj, err := repo.TagObject(r.Hash())
		if err == nil {
			// Annotated tag - get the commit it points to
			commit, err := tagObj.Commit()
			if err == nil {
				return commit.Hash, nil
			}
		}
		// Lightweight tag or failed to get annotated tag target
		return r.Hash(), nil
	}

	// Try as a full reference name
	if r, err := repo.Reference(plumbing.ReferenceName(ref), true); err == nil {
		return r.Hash(), nil
	}

	return plumbing.ZeroHash, fmt.Errorf("unable to resolve ref: %s", ref)
}

// isBinaryContent checks if the content appears to be binary
func isBinaryContent(content []byte) bool {
	// Check for null bytes (common in binary files)
	if bytes.Contains(content[:min(len(content), 8000)], []byte{0}) {
		return true
	}

	// Use http.DetectContentType to check MIME type
	contentType := http.DetectContentType(content)
	if strings.HasPrefix(contentType, "text/") || contentType == "application/json" || contentType == "application/xml" {
		return false
	}

	// Check if content is valid UTF-8
	if !utf8.Valid(content) {
		return true
	}

	return false
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetBlame returns blame information for a file at a given ref
func (g *GitOperations) GetBlame(ctx context.Context, repoPath, ref, filePath string) ([]service.BlameLine, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Resolve the ref to a commit hash
	hash, err := g.resolveRef(repo, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve ref '%s': %w", ref, err)
	}

	// Get the commit
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	// Clean up file path
	filePath = strings.TrimPrefix(filePath, "/")

	// Get blame result using go-git's blame functionality
	blameResult, err := git.Blame(commit, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get blame for '%s': %w", filePath, err)
	}

	// Convert blame result to our BlameLine format
	blameLines := make([]service.BlameLine, len(blameResult.Lines))
	for i, line := range blameResult.Lines {
		blameLines[i] = service.BlameLine{
			LineNo:  i + 1,
			Commit:  line.Hash.String(),
			Author:  line.AuthorName,
			Email:   line.Author,
			Date:    line.Date,
			Content: line.Text,
		}
	}

	return blameLines, nil
}

// GetDiff returns the diff (patch) for a specific commit
func (g *GitOperations) GetDiff(ctx context.Context, repoPath, commitHash string) (*service.DiffResult, error) {
	// Get the raw diff content
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "show", "--format=", "-p", commitHash)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get diff for commit %s: %w (stderr: %s)", commitHash, err, stderr.String())
	}

	content := stdout.String()

	// Get diff stats (files changed, additions, deletions)
	statsCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "show", "--format=", "--stat", "--numstat", commitHash)
	var statsStdout, statsStderr bytes.Buffer
	statsCmd.Stdout = &statsStdout
	statsCmd.Stderr = &statsStderr

	var filesChanged, additions, deletions int
	var files []service.DiffFile

	if err := statsCmd.Run(); err == nil {
		// Parse numstat output for detailed file stats
		// Format: additions<tab>deletions<tab>filename
		lines := strings.Split(statsStdout.String(), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// Skip non-numstat lines (the --stat summary lines)
			parts := strings.Split(line, "\t")
			if len(parts) >= 3 {
				add, _ := strconv.Atoi(parts[0])
				del, _ := strconv.Atoi(parts[1])
				filePath := parts[2]

				additions += add
				deletions += del
				filesChanged++

				// Determine file status
				status := "modified"
				oldPath := filePath
				newPath := filePath

				// Check for renames (format: old => new)
				if strings.Contains(filePath, " => ") {
					status = "renamed"
					renameParts := strings.Split(filePath, " => ")
					if len(renameParts) == 2 {
						oldPath = strings.TrimSpace(renameParts[0])
						newPath = strings.TrimSpace(renameParts[1])
					}
				}

				files = append(files, service.DiffFile{
					OldPath:   oldPath,
					NewPath:   newPath,
					Status:    status,
					Additions: add,
					Deletions: del,
				})
			}
		}
	}

	// If we couldn't parse stats, try to determine from the diff content
	if filesChanged == 0 && content != "" {
		// Count files from "diff --git" lines
		diffLines := strings.Split(content, "\n")
		for _, line := range diffLines {
			if strings.HasPrefix(line, "diff --git") {
				filesChanged++
			} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				additions++
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				deletions++
			}
		}
	}

	// Parse individual file patches from the content
	if len(files) > 0 && content != "" {
		files = g.parseFilePatchesFromDiff(content, files)
	}

	return &service.DiffResult{
		CommitHash:   commitHash,
		Content:      content,
		FilesChanged: filesChanged,
		Additions:    additions,
		Deletions:    deletions,
		Files:        files,
	}, nil
}

// GetCompareDiff returns the diff between two commits
func (g *GitOperations) GetCompareDiff(ctx context.Context, repoPath, from, to string) (*service.DiffResult, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "diff", "--format=", "-p", from+".."+to)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get compare diff %s..%s: %w (stderr: %s)", from, to, err, stderr.String())
	}
	content := stdout.String()

	statsCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "diff", "--format=", "--stat", "--numstat", from+".."+to)
	var statsStdout, statsStderr bytes.Buffer
	statsCmd.Stdout = &statsStdout
	statsCmd.Stderr = &statsStderr

	var filesChanged, additions, deletions int
	var files []service.DiffFile

	if err := statsCmd.Run(); err == nil {
		lines := strings.Split(statsStdout.String(), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Split(line, "\t")
			if len(parts) >= 3 {
				add, _ := strconv.Atoi(parts[0])
				del, _ := strconv.Atoi(parts[1])
				filePath := parts[2]
				additions += add
				deletions += del
				filesChanged++
				status := "modified"
				oldPath := filePath
				newPath := filePath
				if strings.Contains(filePath, " => ") {
					status = "renamed"
					renameParts := strings.Split(filePath, " => ")
					if len(renameParts) == 2 {
						oldPath = strings.TrimSpace(renameParts[0])
						newPath = strings.TrimSpace(renameParts[1])
					}
				}
				files = append(files, service.DiffFile{
					OldPath:   oldPath,
					NewPath:   newPath,
					Status:    status,
					Additions: add,
					Deletions: del,
				})
			}
		}
	}

	if filesChanged == 0 && content != "" {
		diffLines := strings.Split(content, "\n")
		for _, line := range diffLines {
			if strings.HasPrefix(line, "diff --git") {
				filesChanged++
			} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				additions++
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				deletions++
			}
		}
	}

	if len(files) > 0 && content != "" {
		files = g.parseFilePatchesFromDiff(content, files)
	}

	return &service.DiffResult{
		CommitHash:   from + ".." + to,
		Content:      content,
		FilesChanged: filesChanged,
		Additions:    additions,
		Deletions:    deletions,
		Files:        files,
	}, nil
}

// parseFilePatchesFromDiff extracts individual file patches from the full diff content
func (g *GitOperations) parseFilePatchesFromDiff(content string, files []service.DiffFile) []service.DiffFile {
	// Split content by "diff --git" to get individual file diffs
	parts := strings.Split(content, "diff --git ")

	for i, part := range parts {
		if i == 0 || part == "" {
			continue
		}

		// Re-add the prefix that was removed by split
		patch := "diff --git " + part

		// Find which file this patch belongs to
		for j := range files {
			// Check if this patch is for this file
			if strings.Contains(patch, files[j].NewPath) || strings.Contains(patch, files[j].OldPath) {
				files[j].Patch = strings.TrimSpace(patch)

				// Detect added/deleted files from the patch
				if strings.Contains(patch, "new file mode") {
					files[j].Status = "added"
				} else if strings.Contains(patch, "deleted file mode") {
					files[j].Status = "deleted"
				}
				break
			}
		}
	}

	return files
}

// Verify interface compliance at compile time
var _ service.GitService = (*GitOperations)(nil)
