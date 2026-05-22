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
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// UpdateResult contains statistics about the operation
type UpdateResult struct {
	Updated   []string
	Skipped   []string
	Failed    []string
	TotalTime string
}
type DeleteMergedBranchesResult struct {
	Deleted []string
	DryRun  []string
	Skipped []string
	Failed  []string
}

// GitService defines operations for git repositories
type GitService interface {
	UpdateAllBranchesProject(ctx context.Context, path string) (*UpdateResult, error)
	UpdateRemote(ctx context.Context, path string, newRemote string, force bool) error
	FetchLatest(ctx context.Context, path string) error
	DeleteMergedBranches(ctx context.Context, repoPath string, endDate string, dryRun bool) ([]DeleteMergedBranchesResult, error)
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
func (gs *GitModelService) UpdateRemote(ctx context.Context, repoPath string, newRemote string, force bool) error {
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
		if force {
			gs.logger.Warn("fetch failed but force flag is set, skipping rollback", "error", err)
			return nil
		}
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
	// Extract the repository path from the old remote
	repoPath := oldRemote
	if strings.Contains(oldRemote, ":") && !strings.Contains(oldRemote, "://") {
		parts := strings.SplitN(oldRemote, ":", 2)
		repoPath = parts[1]
	}

	// Extract the project name from the repository path
	lastSeparator := strings.LastIndexAny(repoPath, "/\\")
	projectName := repoPath
	if lastSeparator != -1 {
		projectName = repoPath[lastSeparator+1:]
	}
	projectName = strings.TrimSuffix(projectName, ".git")

	// Construct the new remote URL
	if strings.Contains(newRemote, ":") && !strings.Contains(newRemote, "://") {
		return newRemote + "/" + projectName + ".git"
	}
	return path.Join(newRemote, projectName+".git")
}

func (gs *GitModelService) DeleteMergedBranches(ctx context.Context, repoPath string, endDate string, dryRun bool) ([]DeleteMergedBranchesResult, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if repoPath == "" {
		return nil, fmt.Errorf("repository path cannot be empty")
	}
	if endDate == "" {
		return nil, fmt.Errorf("end date cannot be empty")
	}

	cutoff, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date %q, expected YYYY-MM-DD: %w", endDate, err)
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	if err := gs.fetch(ctx, repo); err != nil {
		return nil, err
	}

	featureResults := &DeleteMergedBranchesResult{
		Deleted: []string{},
		DryRun:  []string{},
		Skipped: []string{},
		Failed:  []string{},
	}

	remoteBranches, err := gs.remoteBranches(repo, "origin")
	if err != nil {
		return nil, err
	}

	releaseBranches := filterRemoteBranches(remoteBranches, "origin/release/")
	featureBranches := filterRemoteBranches(remoteBranches, "origin/feature/")
	bugfixBranches := filterRemoteBranches(remoteBranches, "origin/bugfix/")
	hotfixBranches := filterRemoteBranches(remoteBranches, "origin/hotfix/")

	featureResults, err = gs.deleteMergedBranches(ctx, featureBranches, repo, releaseBranches, cutoff, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to delete feature merged branches: %w", err)
	}
	bugfixResults, err := gs.deleteMergedBranches(ctx, bugfixBranches, repo, releaseBranches, cutoff, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to delete bugfix merged branches: %w", err)
	}
	hotfixResults, err := gs.deleteMergedBranches(ctx, hotfixBranches, repo, releaseBranches, cutoff, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to delete hotfix merged branches: %w", err)
	}

	result := make([]DeleteMergedBranchesResult, 4)
	result[0] = *featureResults
	result[1] = *bugfixResults
	result[2] = *hotfixResults

	return result, nil
}

func (gs *GitModelService) deleteMergedBranches(ctx context.Context, featureBranches []string, repo *git.Repository, releaseBranches []string, cutoff time.Time, dryRun bool) (*DeleteMergedBranchesResult, error) {
	result := &DeleteMergedBranchesResult{
		Deleted: []string{},
		DryRun:  []string{},
		Skipped: []string{},
		Failed:  []string{},
	}
	for _, featureBranch := range featureBranches {

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		mergedAt, mergedInto, ok, err := gs.findMergedIntoReleaseDate(repo, featureBranch, releaseBranches)
		if err != nil {
			result.Failed = append(result.Failed, featureBranch)
			gs.logger.Error("failed to inspect branch", "branch", featureBranch, "error", err)
			continue
		}

		if !ok {
			result.Skipped = append(result.Skipped, featureBranch)
			continue
		}

		if mergedAt.After(cutoff) {
			gs.logger.Debug("branch merged after cutoff date",
				"branch", featureBranch,
				"merged_at", mergedAt.Format(time.RFC3339),
				"merged_into", mergedInto)
			result.Skipped = append(result.Skipped, featureBranch)
			continue
		}

		remoteBranchName := strings.TrimPrefix(featureBranch, "origin/")

		if dryRun {
			gs.logger.Info("dry-run: would delete remote branch",
				"branch", remoteBranchName,
				"merged_at", mergedAt.Format("2006-01-02"),
				"merged_into", mergedInto)
			result.DryRun = append(result.DryRun, remoteBranchName)
			continue
		}

		if err := gs.deleteRemoteBranch(repo, "origin", remoteBranchName); err != nil {
			result.Failed = append(result.Failed, remoteBranchName)
			gs.logger.Error("failed to delete remote branch", "branch", remoteBranchName, "error", err)
			continue
		}

		gs.logger.Info("deleted remote branch",
			"branch", remoteBranchName,
			"merged_at", mergedAt.Format("2006-01-02"),
			"merged_into", mergedInto)

		result.Deleted = append(result.Deleted, remoteBranchName)
	}
	return result, nil
}

func (gs *GitModelService) remoteBranches(repo *git.Repository, remoteName string) ([]string, error) {
	refs, err := repo.References()
	if err != nil {
		return nil, fmt.Errorf("failed to list references: %w", err)
	}

	prefix := fmt.Sprintf("refs/remotes/%s/", remoteName)
	branches := []string{}

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if !ref.Name().IsRemote() {
			return nil
		}

		fullName := ref.Name().String()
		if !strings.HasPrefix(fullName, prefix) {
			return nil
		}

		shortName := ref.Name().Short()
		if strings.HasSuffix(shortName, "/HEAD") {
			return nil
		}

		branches = append(branches, shortName)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate references: %w", err)
	}

	return branches, nil
}

func filterRemoteBranches(branches []string, prefix string) []string {
	filtered := []string{}

	for _, branch := range branches {
		if strings.HasPrefix(branch, prefix) {
			filtered = append(filtered, branch)
		}
	}

	return filtered
}

func (gs *GitModelService) findMergedIntoReleaseDate(repo *git.Repository, featureBranch string, releaseBranches []string) (time.Time, string, bool, error) {
	featureRef, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", strings.TrimPrefix(featureBranch, "origin/")), true)
	if err != nil {
		return time.Time{}, "", false, fmt.Errorf("failed to resolve feature branch %s: %w", featureBranch, err)
	}

	featureCommit, err := repo.CommitObject(featureRef.Hash())
	if err != nil {
		return time.Time{}, "", false, fmt.Errorf("failed to load feature commit %s: %w", featureBranch, err)
	}

	var oldestMergeDate time.Time
	var oldestReleaseBranch string
	found := false

	for _, releaseBranch := range releaseBranches {
		releaseRef, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", strings.TrimPrefix(releaseBranch, "origin/")), true)
		if err != nil {
			return time.Time{}, "", false, fmt.Errorf("failed to resolve release branch %s: %w", releaseBranch, err)
		}

		releaseCommit, err := repo.CommitObject(releaseRef.Hash())
		if err != nil {
			return time.Time{}, "", false, fmt.Errorf("failed to load release commit %s: %w", releaseBranch, err)
		}

		merged, err := featureCommit.IsAncestor(releaseCommit)
		if err != nil {
			return time.Time{}, "", false, fmt.Errorf("failed to check ancestry for %s into %s: %w", featureBranch, releaseBranch, err)
		}
		if !merged {
			continue
		}

		mergeDate, ok, err := gs.firstParentMergeDate(featureCommit, releaseCommit)
		if err != nil {
			return time.Time{}, "", false, err
		}
		if !ok {
			continue
		}

		if !found || mergeDate.Before(oldestMergeDate) {
			oldestMergeDate = mergeDate
			oldestReleaseBranch = releaseBranch
			found = true
		}
	}

	return oldestMergeDate, oldestReleaseBranch, found, nil
}

func (gs *GitModelService) firstParentMergeDate(featureCommit *object.Commit, releaseHead *object.Commit) (time.Time, bool, error) {
	firstParentChain := []*object.Commit{}

	current := releaseHead
	for {
		firstParentChain = append(firstParentChain, current)

		if current.NumParents() == 0 {
			break
		}

		parent, err := current.Parent(0)
		if err != nil {
			return time.Time{}, false, fmt.Errorf("failed to read first parent: %w", err)
		}

		current = parent
	}

	for left, right := 0, len(firstParentChain)-1; left < right; left, right = left+1, right-1 {
		firstParentChain[left], firstParentChain[right] = firstParentChain[right], firstParentChain[left]
	}

	for _, commit := range firstParentChain {
		isAncestor, err := featureCommit.IsAncestor(commit)
		if err != nil {
			return time.Time{}, false, fmt.Errorf("failed to check ancestry: %w", err)
		}

		if isAncestor {
			return commit.Committer.When, true, nil
		}
	}

	return time.Time{}, false, nil
}

func (gs *GitModelService) deleteRemoteBranch(repo *git.Repository, remoteName string, branchName string) error {
	refName := plumbing.NewBranchReferenceName(branchName)

	err := repo.Push(&git.PushOptions{
		RemoteName: remoteName,
		RefSpecs: []config.RefSpec{
			config.RefSpec(":" + refName.String()),
		},
	})

	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("failed to delete remote branch %s: %w", branchName, err)
	}

	return nil
}
