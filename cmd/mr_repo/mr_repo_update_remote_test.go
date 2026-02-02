package mr_repo

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/nanaki-93/goktor/cmd"
)

// setupTestDir creates a directory with test repositories
func setupTestDir(t *testing.T, nRepos int) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "goktor-cmd-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create n repositories
	for i := 1; i <= nRepos; i++ {
		repoName := fmt.Sprintf("test-repo-%d", i)
		repoPath := filepath.Join(tmpDir, repoName)

		if err := os.Mkdir(repoPath, 0755); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to create repo dir: %v", err)
		}

		repo, err := git.PlainInit(repoPath, false)
		if err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to init repo: %v", err)
		}

		worktree, err := repo.Worktree()
		if err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to get worktree: %v", err)
		}

		// Create initial file
		testFile := filepath.Join(repoPath, "test.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to write test file: %v", err)
		}

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

		// Add a remote
		if _, err := repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{"https://github.com/oldorg/project.git"},
		}); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to create remote: %v", err)
		}
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestUpdateRemoteCmd(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError bool
		setup     func(*testing.T) (string, func())
	}{
		{
			name:      "missing new-remote flag",
			args:      []string{"mr-repo", "update-remote"},
			wantError: true,
			setup: func(t *testing.T) (string, func()) {
				return setupTestDir(t, 2)
			},
		},
		{
			name:      "empty new-remote value",
			args:      []string{"mr-repo", "update-remote", "--new-remote", ""},
			wantError: true,
			setup: func(t *testing.T) (string, func()) {
				return setupTestDir(t, 2)
			},
		},
		{
			name:      "valid new-remote",
			args:      []string{"mr-repo", "update-remote", "--new-remote", "https://github.com/neworg"},
			wantError: true, // Expected to fail because remotes don't actually exist
			setup: func(t *testing.T) (string, func()) {
				return setupTestDir(t, 2)
			},
		},
		{
			name:      "short flag -a",
			args:      []string{"mr-repo", "update-remote", "-a", "https://github.com/neworg"},
			wantError: true, // Expected to fail because remotes don't actually exist
			setup: func(t *testing.T) (string, func()) {
				return setupTestDir(t, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir, cleanup := tt.setup(t)
			defer cleanup()

			// Change to test directory
			originalWd, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get current dir: %v", err)
			}
			defer os.Chdir(originalWd)

			if err := os.Chdir(testDir); err != nil {
				t.Fatalf("failed to change dir: %v", err)
			}

			cmd.RootCmd.SetArgs(tt.args)

			err = cmd.RootCmd.Execute()

			if (err != nil) != tt.wantError {
				t.Errorf("Execute() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
