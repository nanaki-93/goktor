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
			service := NewFileService()

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

	service := NewFileService()
	result, err := service.ListDirectoriesWithFilter(tmpDir, func(d model.Directory) bool { return true })

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	flatResult := ReorderDirectory(result)
	if len(flatResult) != 16 {
		t.Errorf("got %d directories, want at least 15", len(flatResult))
	}
}

// TestDeepConcurrentTraversalNoDeadlock verifies shared semaphore does not deadlock on deep trees
func TestFileSystemService_DeepConcurrentTraversalNoDeadlock(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 12 top-level directories (> maxWorkers), each with 3 nested subdirectories
	for i := 1; i <= 12; i++ {
		parent := filepath.Join(tmpDir, "parent"+strconv.Itoa(i))
		for j := 1; j <= 3; j++ {
			child := filepath.Join(parent, "child"+strconv.Itoa(j), "leaf")
			if err := os.MkdirAll(child, 0755); err != nil {
				t.Fatalf("failed to create directory %s: %v", child, err)
			}
			if err := os.WriteFile(filepath.Join(child, "data.txt"), []byte("abc"), 0644); err != nil {
				t.Fatalf("failed to create file in %s: %v", child, err)
			}
		}
	}

	// Also exercise lazy semaphore initialization on a struct literal
	service := &FileSystemService{logger: &DefaultLogger{}}
	result, err := service.ListDirectories(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1 root + 12 parents + 36 children + 36 leaves = 85 directories
	flatResult := ReorderDirectory(result)
	if len(flatResult) != 85 {
		t.Errorf("got %d directories, want 85", len(flatResult))
	}
}

// TestRelativePathResolution verifies relative paths are resolved to clean absolute paths
func TestFileSystemService_RelativePathResolution(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "relsub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	rootFile := filepath.Join(tmpDir, "root.txt")
	subFile := filepath.Join(subDir, "nested.txt")
	if err := os.WriteFile(rootFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write root file: %v", err)
	}
	if err := os.WriteFile(subFile, []byte("world!"), 0644); err != nil {
		t.Fatalf("failed to write nested file: %v", err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})

	service := NewFileService()

	dirResult, err := service.ListDirectories(".")
	if err != nil {
		t.Fatalf("ListDirectories('.') failed: %v", err)
	}
	if !filepath.IsAbs(dirResult.FullPath) {
		t.Errorf("expected absolute root FullPath, got %q", dirResult.FullPath)
	}
	if dirResult.Name != filepath.Base(dirResult.FullPath) || dirResult.Name == "." {
		t.Errorf("expected resolved directory base name, got %q", dirResult.Name)
	}
	for _, f := range dirResult.Files {
		if !filepath.IsAbs(f.FullPath) {
			t.Errorf("expected absolute file FullPath, got %q", f.FullPath)
		}
	}
	for _, sub := range dirResult.SubDirs {
		if !filepath.IsAbs(sub.FullPath) {
			t.Errorf("expected absolute subdir FullPath, got %q", sub.FullPath)
		}
		for _, sf := range sub.Files {
			if !filepath.IsAbs(sf.FullPath) {
				t.Errorf("expected absolute nested file FullPath, got %q", sf.FullPath)
			}
		}
	}

	files, err := service.ListFiles("relsub")
	if err != nil {
		t.Fatalf("ListFiles('relsub') failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if !filepath.IsAbs(files[0].FullPath) {
		t.Errorf("expected absolute file path from ListFiles, got %q", files[0].FullPath)
	}
	if filepath.Base(files[0].FullPath) != "nested.txt" {
		t.Errorf("expected nested.txt base name, got %q", filepath.Base(files[0].FullPath))
	}
}

// TestGetDirectoryRecursivelyWithErrors tests error handling in recursive calls
func TestFileSystemService_RecursiveErrorHandling(t *testing.T) {

	service := NewFileService()
	_, err := service.ListDirectoriesWithFilter("/nonexistent/path", func(d model.Directory) bool { return true })

	if err == nil {
		t.Error("expected error for non-existent path, got nil")
	}
}
