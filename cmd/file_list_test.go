package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileListCmd(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		args    []string
		wantErr bool
	}{
		{
			name: "list files with default directory",
			setup: func(t *testing.T) string {
				return ""
			},
			args:    []string{"file-list"},
			wantErr: false,
		},
		{
			name: "list files in directory",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content"), 0644))
				return tmpDir
			},
			args:    []string{"file-list", "-d"},
			wantErr: false,
		},
		{
			name: "nonexistent directory",
			setup: func(t *testing.T) string {
				return "/nonexistent/path"
			},
			args:    []string{"file-list", "-d", "/nonexistent/path"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := tt.setup(t)

			output := &bytes.Buffer{}
			rootCmd.SetOut(output)
			rootCmd.SetErr(output)

			var finalArgs []string
			finalArgs = append(finalArgs, tt.args...)
			if tmpDir != "" && tt.args[0] == "file-list" && len(tt.args) > 1 && tt.args[1] == "-d" {
				finalArgs = append(finalArgs, tmpDir)
			}

			rootCmd.SetArgs(finalArgs)

			err := rootCmd.Execute()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
