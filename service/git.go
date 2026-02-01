package service

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// UpdateResult contains statistics about the operation
type UpdateResult struct {
	Updated   []string
	Skipped   []string
	Failed    []string
	TotalTime string
}

// GitService defines operations for git repositories
type GitService interface {
	// UpdateAllBranchesProject aligns all local branches with remote (except current branch)
	UpdateAllBranchesProject(ctx context.Context, path string) (*UpdateResult, error)

	// UpdateRemote changes the origin remote URL and verifies connectivity
	UpdateRemote(ctx context.Context, path string, newRemote string) error

	// FetchLatest fetches latest updates from remote without modifying branches
	FetchLatest(ctx context.Context, path string) error
}

// GitModelService implements GitService
type GitModelService struct {
	logger Logger
}

// Logger interface for flexible logging
type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// NewGitService creates a new git service with default logger
func NewGitService() GitService {
	return &GitModelService{
		logger: &DefaultLogger{},
	}
}

// NewGitServiceWithLogger creates a new git service with custom logger
func NewGitServiceWithLogger(logger Logger) GitService {
	return &GitModelService{
		logger: logger,
	}
}

// FetchLatest fetches latest updates from remote without modifying branches
func (gs *GitModelService) FetchLatest(ctx context.Context, repoPath string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	err = repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName: "origin",
		Force:      true,
		Tags:       git.AllTags,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fetch failed: %w", err)
	}

	gs.logger.Info("fetch completed successfully")
	return nil
}

// UpdateAllBranchesProject aligns all local branches with their remote counterparts
func (gs *GitModelService) UpdateAllBranchesProject(ctx context.Context, repoPath string) (*UpdateResult, error) {
	result := &UpdateResult{
		Updated: []string{},
		Skipped: []string{},
		Failed:  []string{},
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repo: %w", err)
	}

	// Fetch latest updates from remote
	gs.logger.Info("fetching latest updates from remote")
	err = repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName: "origin",
		Force:      true,
		Tags:       git.AllTags,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	// Get current branch to protect it
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	currentBranch := head.Name().Short()
	gs.logger.Info("protecting current branch", "branch", currentBranch)

	branches, err := repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Process each branch
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		branchName := ref.Name().Short()

		// Skip current branch to protect uncommitted changes
		if branchName == currentBranch {
			gs.logger.Debug("skipping current branch", "branch", branchName)
			result.Skipped = append(result.Skipped, branchName)
			return nil
		}

		// Get remote tracking branch
		remoteRef, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", branchName), true)
		if err != nil {
			gs.logger.Warn("remote tracking branch not found", "branch", branchName)
			result.Skipped = append(result.Skipped, branchName)
			return nil
		}

		// Checkout branch
		if err := worktree.Checkout(&git.CheckoutOptions{
			Branch: ref.Name(),
			Force:  false,
		}); err != nil {
			gs.logger.Error("failed to checkout branch", "branch", branchName, "error", err)
			result.Failed = append(result.Failed, branchName)
			return nil
		}

		// Reset to remote
		if err := worktree.Reset(&git.ResetOptions{
			Mode:   git.HardReset,
			Commit: remoteRef.Hash(),
		}); err != nil {
			gs.logger.Error("failed to reset branch", "branch", branchName, "error", err)
			result.Failed = append(result.Failed, branchName)
			return nil
		}

		gs.logger.Info("branch updated", "branch", branchName)
		result.Updated = append(result.Updated, branchName)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed processing branches: %w", err)
	}

	// Checkout back to original branch
	if err := worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(currentBranch),
	}); err != nil {
		return nil, fmt.Errorf("failed to checkout back to %s: %w", currentBranch, err)
	}

	gs.logger.Info("update completed",
		"updated", len(result.Updated),
		"skipped", len(result.Skipped),
		"failed", len(result.Failed))

	return result, nil
}

// UpdateRemote updates the origin remote URL and verifies connectivity
func (gs *GitModelService) UpdateRemote(ctx context.Context, repoPath string, newRemote string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	gs.logger.Info("updating remote", "repo", repoPath)

	remotes, err := repo.Remotes()
	if err != nil {
		return fmt.Errorf("failed to list remotes: %w", err)
	}

	var originRemote *git.Remote
	for _, r := range remotes {
		if r.Config().Name == "origin" {
			originRemote = r
			break
		}
	}

	if originRemote == nil {
		return fmt.Errorf("remote 'origin' not found")
	}

	// Store old remote for rollback
	oldRemote := originRemote.Config().URLs[0]
	gs.logger.Debug("current remote", "url", oldRemote)

	// Update remote URL
	projectName, _, newRemoteURL := parseRemoteURL(newRemote, oldRemote)
	gs.logger.Debug("new remote URL", "url", newRemoteURL, "project", projectName)

	// Update config
	cfg, err := repo.Storer.Config()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	remoteCfg, ok := cfg.Remotes["origin"]
	if !ok {
		return fmt.Errorf("remote 'origin' not found in config")
	}

	remoteCfg.URLs = []string{newRemoteURL}

	if err := repo.Storer.SetConfig(cfg); err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	// Verify connectivity
	gs.logger.Info("verifying remote connectivity")
	err = repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName: "origin",
		Force:      true,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		gs.logger.Error("fetch failed, rolling back", "error", err)

		// Rollback
		remoteCfg.URLs = []string{oldRemote}
		if err := repo.Storer.SetConfig(cfg); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}

		return fmt.Errorf("fetch failed, rollback completed: %w", err)
	}

	gs.logger.Info("remote updated successfully")
	return nil
}

// parseRemoteURL handles both HTTP URLs and local file paths
func parseRemoteURL(newRemote string, oldRemote string) (projectName, oldRemoteBase, newRemoteURL string) {
	isURL := isHTTPRemote(oldRemote)

	if isURL {
		return manageRemoteURL(newRemote, oldRemote)
	}
	return manageRemoteLocal(newRemote, oldRemote)
}

// isHTTPRemote checks if the remote is an HTTP(S) URL
func isHTTPRemote(remote string) bool {
	return strings.HasPrefix(remote, "http://") ||
		strings.HasPrefix(remote, "https://") ||
		strings.HasPrefix(remote, "git://") ||
		strings.HasPrefix(remote, "ssh://")
}

// manageRemoteLocal handles local file path remotes
func manageRemoteLocal(newRemote string, oldRemote string) (string, string, string) {
	projectName := filepath.Base(oldRemote)
	oldRemoteBase := oldRemote[:len(oldRemote)-len(projectName)]
	oldRemoteBase = filepath.Clean(oldRemoteBase)
	return projectName, oldRemoteBase, filepath.Join(newRemote, projectName)
}

// manageRemoteURL handles HTTP(S) URL remotes
func manageRemoteURL(newRemote string, oldRemote string) (string, string, string) {
	projectName := path.Base(oldRemote)
	oldRemoteBase := oldRemote[:len(oldRemote)-len(projectName)]
	if len(oldRemoteBase) > 0 && oldRemoteBase[len(oldRemoteBase)-1] == '/' {
		oldRemoteBase = oldRemoteBase[:len(oldRemoteBase)-1]
	}
	// Remove .git suffix if present
	if strings.HasSuffix(projectName, ".git") {
		projectName = projectName[:len(projectName)-4]
	}
	return projectName, oldRemoteBase, newRemote + "/" + projectName + ".git"
}

// DefaultLogger implements Logger interface using fmt
type DefaultLogger struct{}

func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	fmt.Printf("ℹ [INFO] %s %v\n", msg, args)
}

func (l *DefaultLogger) Warn(msg string, args ...interface{}) {
	fmt.Printf("⚠ [WARN] %s %v\n", msg, args)
}

func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	fmt.Printf("✗ [ERROR] %s %v\n", msg, args)
}

func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	fmt.Printf("🔍 [DEBUG] %s %v\n", msg, args)
}
