package service

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

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
	UpdateAllBranchesProject(ctx context.Context, path string) (*UpdateResult, error)
	UpdateRemote(ctx context.Context, path string, newRemote string) error
	FetchLatest(ctx context.Context, path string) error
}

// GitModelService implements GitService
type GitModelService struct {
	logger Logger
}

// NewGitService creates a new git service with default logger
func NewGitService(logger Logger) GitService {
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

	return gs.fetch(ctx, repo)
}

func (gs *GitModelService) fetch(ctx context.Context, repo *git.Repository) error {
	err := repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName: "origin",
		Force:      true,
		Tags:       git.AllTags,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fetch failed: %w", err)
	}
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
	if err := gs.fetch(ctx, repo); err != nil {
		return nil, err
	}

	currentBranch, err := gs.getCurrentBranch(repo)
	if err != nil {
		return nil, err
	}
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

		if err := gs.updateBranch(repo, worktree, branchName, ref, result); err != nil {
			result.Failed = append(result.Failed, branchName)
			gs.logger.Error("failed to update branch", "branch", branchName, "error", err)
			return nil
		}
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

func (gs *GitModelService) getCurrentBranch(repo *git.Repository) (string, error) {
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}
	return head.Name().Short(), nil
}

// updateBranch updates a single branch
func (gs *GitModelService) updateBranch(repo *git.Repository, worktree *git.Worktree, branchName string, ref *plumbing.Reference, result *UpdateResult) error {
	remoteRef, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", branchName), true)
	if err != nil {
		gs.logger.Warn("remote tracking branch not found", "branch", branchName)
		result.Skipped = append(result.Skipped, branchName)
		return nil
	}

	if err := worktree.Checkout(&git.CheckoutOptions{
		Branch: ref.Name(),
		Force:  false,
	}); err != nil {
		return err
	}

	if err := worktree.Reset(&git.ResetOptions{
		Mode:   git.HardReset,
		Commit: remoteRef.Hash(),
	}); err != nil {
		return err
	}

	gs.logger.Info("branch updated", "branch", branchName)
	result.Updated = append(result.Updated, branchName)
	return nil
}

// UpdateRemote updates the origin remote URL and verifies connectivity
func (gs *GitModelService) UpdateRemote(ctx context.Context, repoPath string, newRemote string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	gs.logger.Debug("updating remote", "repo", repoPath)

	cfg, err := repo.Storer.Config()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	remoteCfg, ok := cfg.Remotes["origin"]
	if !ok {
		return fmt.Errorf("remote 'origin' not found in config")
	}

	oldRemote := remoteCfg.URLs[0]
	newRemoteURL := parseRemoteURL(newRemote, oldRemote)

	gs.logger.Debug("updating remote", "from", oldRemote, "to", newRemoteURL)

	remoteCfg.URLs = []string{newRemoteURL}
	if err := repo.Storer.SetConfig(cfg); err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := gs.fetch(fetchCtx, repo); err != nil {
		remoteCfg.URLs = []string{oldRemote}
		if rollbackErr := repo.Storer.SetConfig(cfg); rollbackErr != nil {
			return fmt.Errorf("fetch failed and rollback failed: fetch=%w, rollback=%w", err, rollbackErr)
		}
		return fmt.Errorf("fetch failed, rollback completed: %w", err)
	}

	gs.logger.Info("remote updated successfully: ", "new remote", newRemoteURL)
	return nil
}

// parseRemoteURL handles both HTTP URLs and local file paths
func parseRemoteURL(newRemote string, oldRemote string) string {
	if isNetworkRemote(oldRemote) {
		return buildNetworkRemote(newRemote, oldRemote)
	}
	return buildLocalRemote(newRemote, oldRemote)
}

// buildLocalRemote constructs local file path remote
func buildLocalRemote(newRemote string, oldRemote string) string {
	projectName := filepath.Base(oldRemote)
	return filepath.Join(newRemote, projectName)
}

func isNetworkRemote(remote string) bool {
	return strings.HasPrefix(remote, "http://") ||
		strings.HasPrefix(remote, "https://") ||
		strings.HasPrefix(remote, "git://") ||
		strings.HasPrefix(remote, "ssh://") ||
		strings.Contains(remote, "@")
}

// buildNetworkRemote handles HTTP(S) and SSH URL remotes
func buildNetworkRemote(newRemote, oldRemote string) string {
	var projectName string
	var repoPath string

	if strings.Contains(oldRemote, ":") && !strings.Contains(oldRemote, "://") {
		// Handles user@host:path/to/repo.git
		parts := strings.SplitN(oldRemote, ":", 2)
		repoPath = parts[1]
	} else {
		// Handles https://host/path/to/repo.git
		repoPath = oldRemote
	}

	// Find the last separator to get the project name
	lastSlash := strings.LastIndex(repoPath, "/")
	lastBackslash := strings.LastIndex(repoPath, "\\")

	lastSeparator := lastSlash
	if lastBackslash > lastSlash {
		lastSeparator = lastBackslash
	}

	if lastSeparator != -1 {
		projectName = repoPath[lastSeparator+1:]
	} else {
		projectName = repoPath
	}
	projectName = strings.TrimSuffix(projectName, ".git")

	// For SCP-like SSH syntax (e.g., git@host:repo), use string concatenation.
	if strings.Contains(newRemote, ":") && !strings.Contains(newRemote, "://") {
		return newRemote + "/" + projectName + ".git"
	}

	// For HTTP(S) or other URL schemes, use path.Join.
	return path.Join(newRemote, projectName+".git")
}
