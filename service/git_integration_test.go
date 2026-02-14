package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// setupBareRepo creates a bare repository to act as a remote
func setupBareRepo(t *testing.T) (string, func()) {
	t.Helper()

	bareDir, err := os.MkdirTemp("", "goktor-bare-*")
	if err != nil {
		t.Fatalf("failed to create bare dir: %v", err)
	}

	_, err = git.PlainInit(bareDir, true)
	if err != nil {
		os.RemoveAll(bareDir)
		t.Fatalf("failed to init bare repo: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(bareDir)
	}

	return bareDir, cleanup
}

func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "goktor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize git repository
	repo, err := git.PlainInit(tmpDir, false)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to init repo: %v", err)
	}

	// Create initial commit
	worktree, err := repo.Worktree()
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to get worktree: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to write test file: %v", err)
	}

	// Add and commit
	if _, err := worktree.Add("test.txt"); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to add file: %v", err)
	}

	if _, err := worktree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	}); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to commit: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// setupTestRepoWithRemote creates a test repo with a local bare repo as remote
func setupTestRepoWithRemote(t *testing.T) (string, string, func()) {
	t.Helper()

	// Create bare repo first
	bareDir, bareCleanup := setupBareRepo(t)

	// Create working repo
	tmpDir, err := os.MkdirTemp("", "goktor-test-*")
	if err != nil {
		bareCleanup()
		t.Fatalf("failed to create temp dir: %v", err)
	}

	repo, err := git.PlainInit(tmpDir, false)
	if err != nil {
		bareCleanup()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to init repo: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		bareCleanup()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to get worktree: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		bareCleanup()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to write test file: %v", err)
	}

	if _, err := worktree.Add("test.txt"); err != nil {
		bareCleanup()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to add file: %v", err)
	}

	if _, err := worktree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	}); err != nil {
		bareCleanup()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to commit: %v", err)
	}

	// Add origin remote pointing to bare repo
	if _, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{bareDir},
	}); err != nil {
		bareCleanup()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create remote: %v", err)
	}

	// Push to bare repo
	if err := repo.Push(&git.PushOptions{
		RemoteName: "origin",
	}); err != nil {
		bareCleanup()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to push: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
		bareCleanup()
	}

	return tmpDir, bareDir, cleanup
}

func setupTestRepoWithBranches(t *testing.T) (string, string, func()) {
	t.Helper()

	repoPath, bareDir, cleanup := setupTestRepoWithRemote(t)

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		cleanup()
		t.Fatalf("failed to open repo: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		cleanup()
		t.Fatalf("failed to get worktree: %v", err)
	}

	// Create additional branches
	head, err := repo.Head()
	if err != nil {
		cleanup()
		t.Fatalf("failed to get HEAD: %v", err)
	}

	// Create feature branch
	featureRef := plumbing.NewHashReference("refs/heads/feature", head.Hash())
	if err := repo.Storer.SetReference(featureRef); err != nil {
		cleanup()
		t.Fatalf("failed to create feature branch: %v", err)
	}

	// Create develop branch
	developRef := plumbing.NewHashReference("refs/heads/develop", head.Hash())
	if err := repo.Storer.SetReference(developRef); err != nil {
		cleanup()
		t.Fatalf("failed to create develop branch: %v", err)
	}

	// Push feature and develop branches to bare repo
	repo.Push(&git.PushOptions{RemoteName: "origin"})

	// Create a second commit on main
	testFile := filepath.Join(repoPath, "test2.txt")
	if err := os.WriteFile(testFile, []byte("second test content"), 0644); err != nil {
		cleanup()
		t.Fatalf("failed to write second test file: %v", err)
	}

	if _, err := worktree.Add("test2.txt"); err != nil {
		cleanup()
		t.Fatalf("failed to add second file: %v", err)
	}

	if _, err := worktree.Commit("second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	}); err != nil {
		cleanup()
		t.Fatalf("failed to second commit: %v", err)
	}

	// Push updated main branch
	repo.Push(&git.PushOptions{RemoteName: "origin"})

	return repoPath, bareDir, cleanup
}

// TestFetchLatest tests the FetchLatest method
func TestGitModelService_FetchLatest(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testing.T) (string, func())
		wantErr bool
	}{
		{
			name: "successful fetch",
			setup: func(t *testing.T) (string, func()) {
				repoPath, _, cleanup := setupTestRepoWithRemote(t)
				return repoPath, cleanup
			},
			wantErr: false,
		},
		{
			name: "non-existent path",
			setup: func(t *testing.T) (string, func()) {
				return "/non/existent/path", func() {}
			},
			wantErr: true,
		},
		{
			name: "invalid repository",
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "goktor-invalid-*")
				if err != nil {
					t.Fatalf("failed to create temp dir: %v", err)
				}
				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath, cleanup := tt.setup(t)
			defer cleanup()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			service := NewGitService(&DefaultLogger{})
			err := service.FetchLatest(ctx, repoPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("FetchLatest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestUpdateAllBranchesProject tests the UpdateAllBranchesProject method
func TestGitModelService_UpdateAllBranchesProject(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*testing.T) (string, func())
		wantErr     bool
		wantUpdated int
		wantSkipped int
	}{
		{
			name: "single branch repository",
			setup: func(t *testing.T) (string, func()) {
				repoPath, _, cleanup := setupTestRepoWithRemote(t)
				return repoPath, cleanup
			},
			wantErr:     false,
			wantUpdated: 0, // Only main exists, but it's the current branch so skipped
			wantSkipped: 1, // main is skipped (current branch)
		},
		{
			name: "repository with multiple branches",
			setup: func(t *testing.T) (string, func()) {
				repoPath, _, cleanup := setupTestRepoWithBranches(t)
				return repoPath, cleanup
			},
			wantErr:     false,
			wantUpdated: 2, // feature and develop branches should be updated
			wantSkipped: 1, // main is skipped (current branch)
		},
		{
			name: "non-existent path",
			setup: func(t *testing.T) (string, func()) {
				return "/non/existent/path", func() {}
			},
			wantErr: true,
		},
		{
			name: "invalid repository",
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "goktor-invalid-*")
				if err != nil {
					t.Fatalf("failed to create temp dir: %v", err)
				}
				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath, cleanup := tt.setup(t)
			defer cleanup()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			service := NewGitService(&DefaultLogger{})
			result, err := service.UpdateAllBranchesProject(ctx, repoPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateAllBranchesProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Verify result
			if len(result.Updated) != tt.wantUpdated {
				t.Errorf("Updated branches = %d, want %d. Branches: %v", len(result.Updated), tt.wantUpdated, result.Updated)
			}

			if len(result.Skipped) != tt.wantSkipped {
				t.Errorf("Skipped branches = %d, want %d. Branches: %v", len(result.Skipped), tt.wantSkipped, result.Skipped)
			}

			if len(result.Failed) > 0 {
				t.Errorf("Failed branches = %v", result.Failed)
			}
		})
	}
}

// TestContextCancellation tests that operations can be cancelled via context
func TestGitModelService_ContextCancellation(t *testing.T) {
	repoPath, _, cleanup := setupTestRepoWithBranches(t)
	defer cleanup()

	// Create context that is already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	service := NewGitService(&DefaultLogger{})
	_, err := service.UpdateAllBranchesProject(ctx, repoPath)

	if err == nil {
		t.Error("Expected error from cancelled context, got nil")
	}
}

// TestUpdateRemote tests the UpdateRemote method
func TestGitModelService_UpdateRemote(t *testing.T) {
	t.Run("remote update preserves project name", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "goktor-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		repo, _ := git.PlainInit(tmpDir, false)
		worktree, _ := repo.Worktree()

		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test content"), 0644)
		worktree.Add("test.txt")
		worktree.Commit("initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test User",
				Email: "test@example.com",
				When:  time.Now(),
			},
		})

		// Add HTTP remote (typical use case)
		repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{"https://github.com/oldorg/my-project.git"},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := NewGitService(&DefaultLogger{})
		err = service.UpdateRemote(ctx, tmpDir, "https://github.com/neworg")

		// Should error because fetch will fail (remote doesn't exist)
		if err == nil {
			t.Error("UpdateRemote() expected error for non-existent remote, got nil")
		}

		// Check if project name was preserved (after rollback due to failed fetch)
		repo, _ = git.PlainOpen(tmpDir)
		cfg, _ := repo.Storer.Config()
		currentURL := cfg.Remotes["origin"].URLs[0]

		// After rollback, should be back to original
		expectedURL := "https://github.com/oldorg/my-project.git"
		if currentURL != expectedURL {
			t.Errorf("After rollback, URL = %v, want %v", currentURL, expectedURL)
		}
	})

	t.Run("repository without origin remote", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "goktor-no-remote-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		repo, err := git.PlainInit(tmpDir, false)
		if err != nil {
			t.Fatalf("failed to init repo: %v", err)
		}

		worktree, _ := repo.Worktree()
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		worktree.Add("test.txt")
		worktree.Commit("initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test User",
				Email: "test@example.com",
				When:  time.Now(),
			},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := NewGitService(&DefaultLogger{})
		err = service.UpdateRemote(ctx, tmpDir, "https://github.com/neworg")

		if err == nil {
			t.Error("UpdateRemote() expected error for missing origin, got nil")
		}
	})

	t.Run("non-existent path", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := NewGitService(&DefaultLogger{})
		err := service.UpdateRemote(ctx, "/non/existent/path", "https://github.com/neworg")

		if err == nil {
			t.Error("UpdateRemote() expected error for non-existent path, got nil")
		}
	})

	t.Run("invalid remote URL causes rollback", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "goktor-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		repo, _ := git.PlainInit(tmpDir, false)
		worktree, _ := repo.Worktree()

		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test content"), 0644)
		worktree.Add("test.txt")
		worktree.Commit("initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test User",
				Email: "test@example.com",
				When:  time.Now(),
			},
		})

		// Add HTTP remote
		originalURL := "https://github.com/testorg/test-project.git"
		repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{originalURL},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := NewGitService(&DefaultLogger{})
		err = service.UpdateRemote(ctx, tmpDir, "https://github.com/nonexistent")

		// Should error because fetch fails
		if err == nil {
			t.Error("UpdateRemote() expected error for invalid remote, got nil")
		}

		// Verify remote was rolled back
		repo, _ = git.PlainOpen(tmpDir)
		cfg, _ := repo.Storer.Config()
		currentURL := cfg.Remotes["origin"].URLs[0]

		if currentURL != originalURL {
			t.Errorf("Remote should be rolled back to %v, got %v", originalURL, currentURL)
		}
	})
}

// TestUpdateRemote_ProjectNamePreservation tests project name preservation
func TestGitModelService_UpdateRemote_ProjectNamePreservation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "goktor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repo, _ := git.PlainInit(tmpDir, false)
	worktree, _ := repo.Worktree()

	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)
	worktree.Add("test.txt")
	worktree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})

	// Add HTTP remote with specific project name
	originalURL := "https://gitlab.com/mycompany/awesome-project.git"
	repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{originalURL},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	service := NewGitService(&DefaultLogger{})
	err = service.UpdateRemote(ctx, tmpDir, "https://github.com/newcompany")

	// Will fail fetch, but check the URL construction logic
	repo, _ = git.PlainOpen(tmpDir)
	cfg, _ := repo.Storer.Config()
	currentURL := cfg.Remotes["origin"].URLs[0]

	// After rollback, should be back to original
	if currentURL != originalURL {
		t.Errorf("After rollback: URL = %v, want %v", currentURL, originalURL)
	}
}

// TestUpdateRemote_LocalFilePaths tests local file path remotes
func TestGitModelService_UpdateRemote_LocalFilePaths(t *testing.T) {
	t.Run("local bare repository with .git suffix", func(t *testing.T) {
		// Create two bare repos
		oldBareDir, oldCleanup := setupBareRepo(t)
		defer oldCleanup()

		newBareDir, newCleanup := setupBareRepo(t)
		defer newCleanup()

		// Create working repo
		tmpDir, err := os.MkdirTemp("", "goktor-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		repo, _ := git.PlainInit(tmpDir, false)
		worktree, _ := repo.Worktree()

		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test content"), 0644)
		worktree.Add("test.txt")
		worktree.Commit("initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test User",
				Email: "test@example.com",
				When:  time.Now(),
			},
		})

		// Add local file path remote with .git suffix
		originalRemote := oldBareDir + "/my-project.git"
		repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{originalRemote},
		})

		// Push to old bare repo to make it valid
		repo.Push(&git.PushOptions{RemoteName: "origin"})

		// Create a bare repo at the expected new location
		newRepoPath := filepath.Join(newBareDir, "my-project.git")
		git.PlainInit(newRepoPath, true)

		// Push to new bare repo
		repo.DeleteRemote("origin")
		repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{newRepoPath},
		})
		repo.Push(&git.PushOptions{RemoteName: "origin"})

		// Switch back to old remote
		repo.DeleteRemote("origin")
		repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{originalRemote},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := NewGitService(&DefaultLogger{})
		err = service.UpdateRemote(ctx, tmpDir, newBareDir)

		if err != nil {
			t.Errorf("UpdateRemote() should succeed with valid local paths, got error: %v", err)
		}

		repo, _ = git.PlainOpen(tmpDir)
		cfg, _ := repo.Storer.Config()
		currentURL := cfg.Remotes["origin"].URLs[0]

		expectedURL := filepath.Join(newBareDir, "my-project.git")
		if currentURL != expectedURL {
			t.Errorf("Remote URL = %v, want %v", currentURL, expectedURL)
		}
	})

	t.Run("local path without .git suffix", func(t *testing.T) {
		oldBareDir, oldCleanup := setupBareRepo(t)
		defer oldCleanup()

		newBareDir, newCleanup := setupBareRepo(t)
		defer newCleanup()

		tmpDir, err := os.MkdirTemp("", "goktor-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		repo, _ := git.PlainInit(tmpDir, false)
		worktree, _ := repo.Worktree()

		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test content"), 0644)
		worktree.Add("test.txt")
		worktree.Commit("initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test User",
				Email: "test@example.com",
				When:  time.Now(),
			},
		})

		repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{oldBareDir},
		})

		repo.Push(&git.PushOptions{RemoteName: "origin"})

		oldBaseName := filepath.Base(oldBareDir)
		newRepoPath := filepath.Join(newBareDir, oldBaseName)
		git.PlainInit(newRepoPath, true)

		repo.DeleteRemote("origin")
		repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{newRepoPath},
		})
		repo.Push(&git.PushOptions{RemoteName: "origin"})

		repo.DeleteRemote("origin")
		repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{oldBareDir},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := NewGitService(&DefaultLogger{})
		err = service.UpdateRemote(ctx, tmpDir, newBareDir)

		if err != nil {
			t.Errorf("UpdateRemote() should succeed, got error: %v", err)
		}

		repo, _ = git.PlainOpen(tmpDir)
		cfg, _ := repo.Storer.Config()
		currentURL := cfg.Remotes["origin"].URLs[0]

		expectedURL := filepath.Join(newBareDir, oldBaseName)
		if currentURL != expectedURL {
			t.Errorf("Remote URL = %v, want %v", currentURL, expectedURL)
		}
	})

	t.Run("windows absolute path", func(t *testing.T) {
		if filepath.Separator != '\\' {
			t.Skip("Skipping Windows-specific test on non-Windows platform")
		}

		oldBareDir, oldCleanup := setupBareRepo(t)
		defer oldCleanup()

		newBareDir, newCleanup := setupBareRepo(t)
		defer newCleanup()

		tmpDir, err := os.MkdirTemp("", "goktor-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		repo, _ := git.PlainInit(tmpDir, false)
		worktree, _ := repo.Worktree()

		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test content"), 0644)
		worktree.Add("test.txt")
		worktree.Commit("initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test User",
				Email: "test@example.com",
				When:  time.Now(),
			},
		})

		originalRemote := oldBareDir + "\\project.git"
		repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{originalRemote},
		})

		repo.Push(&git.PushOptions{RemoteName: "origin"})

		newRepoPath := filepath.Join(newBareDir, "project.git")
		git.PlainInit(newRepoPath, true)

		repo.DeleteRemote("origin")
		repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{newRepoPath},
		})
		repo.Push(&git.PushOptions{RemoteName: "origin"})

		repo.DeleteRemote("origin")
		repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{originalRemote},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := NewGitService(&DefaultLogger{})
		err = service.UpdateRemote(ctx, tmpDir, newBareDir)

		if err != nil {
			t.Errorf("UpdateRemote() should succeed, got error: %v", err)
		}

		repo, _ = git.PlainOpen(tmpDir)
		cfg, _ := repo.Storer.Config()
		currentURL := cfg.Remotes["origin"].URLs[0]

		expectedURL := filepath.Join(newBareDir, "project.git")
		if currentURL != expectedURL {
			t.Errorf("Remote URL = %v, want %v", currentURL, expectedURL)
		}
	})
}
