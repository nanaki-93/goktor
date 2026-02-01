package service

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/nanaki-93/goktor/model"
)

// TestProcessSubDirectoriesRecursively tests the recursive call in processSubDirectories
func TestFileSystemService_ProcessSubDirectoriesRecursively(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T) string
		filter        func(model.Directory) bool
		expectedCount int
		wantErr       bool
	}{
		{
			name: "nested directories with filter",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				os.MkdirAll(filepath.Join(tmpDir, "sub1", "sub2", "sub3"), 0755)
				os.MkdirAll(filepath.Join(tmpDir, "sub1", "sub4"), 0755)
				return tmpDir
			},
			filter:        func(d model.Directory) bool { return true },
			expectedCount: 5,
			wantErr:       false,
		},
		{
			name: "single level subdirectories",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				os.MkdirAll(filepath.Join(tmpDir, "dir1"), 0755)
				os.MkdirAll(filepath.Join(tmpDir, "dir2"), 0755)
				os.MkdirAll(filepath.Join(tmpDir, "dir3"), 0755)
				return tmpDir
			},
			filter:        func(d model.Directory) bool { return true },
			expectedCount: 4,
			wantErr:       false,
		},
		{
			name: "filter excludes directories by size",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				os.MkdirAll(filepath.Join(tmpDir, "large"), 0755)
				os.WriteFile(filepath.Join(tmpDir, "large", "file.txt"), make([]byte, 2*OneGb), 0644)
				os.MkdirAll(filepath.Join(tmpDir, "small"), 0755)
				return tmpDir
			},
			filter:        func(d model.Directory) bool { return d.Size > OneGb },
			expectedCount: 1,
			wantErr:       false,
		},
		{
			name: "permission denied on subdirectory",
			setup: func(t *testing.T) string {
				if os.Geteuid() == 0 {
					t.Skip("Skipping permission test when running as root")
				}
				tmpDir := t.TempDir()
				restrictedDir := filepath.Join(tmpDir, "restricted")
				os.MkdirAll(restrictedDir, 0755)
				os.Chmod(restrictedDir, 0000)

				// Verify the permission actually worked
				_, err := os.ReadDir(restrictedDir)
				if err == nil {
					t.Skip("Skipping permission test - chmod 000 not enforced on this system")
				}

				t.Cleanup(func() { os.Chmod(restrictedDir, 0755) })
				return tmpDir
			},
			filter:        func(d model.Directory) bool { return true },
			expectedCount: 1, // Skipped due to permission error
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := tt.setup(t)
			service := NewService()

			result, err := service.ListDirectoriesWithFilter(tmpDir, tt.filter)

			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				return
			}

			flatResult := ReorderDirectory(result)
			if len(flatResult) != tt.expectedCount {
				t.Errorf("got %d directories, want %d", len(flatResult), tt.expectedCount)
			}
		})
	}
}

// TestConcurrentSubDirectoryProcessing verifies parallel processing works correctly
func TestFileSystemService_ConcurrentSubDirectoryProcessing(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 15 subdirectories to exceed maxWorkers (10)
	for i := 1; i <= 15; i++ {
		dirPath := filepath.Join(tmpDir, "dir"+strconv.Itoa(i))
		os.MkdirAll(dirPath, 0755)
		os.WriteFile(filepath.Join(dirPath, "file.txt"), []byte("content"), 0644)
	}

	service := NewService()
	result, err := service.ListDirectoriesWithFilter(tmpDir, func(d model.Directory) bool { return true })

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	flatResult := ReorderDirectory(result)
	if len(flatResult) != 16 {
		t.Errorf("got %d directories, want at least 15", len(flatResult))
	}
}

// TestGetDirectoryRecursivelyWithErrors tests error handling in recursive calls
func TestFileSystemService_RecursiveErrorHandling(t *testing.T) {

	service := NewService()
	_, err := service.ListDirectoriesWithFilter("/nonexistent/path", func(d model.Directory) bool { return true })

	if err == nil {
		t.Error("expected error for non-existent path, got nil")
	}
}
