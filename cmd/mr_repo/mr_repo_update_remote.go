package mr_repo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nanaki-93/goktor/service"
	"github.com/spf13/cobra"
)

var updateRemoteCmd = &cobra.Command{
	Use:          "update-remote",
	Short:        "Update remote URLs for all repositories",
	Long:         `Update the remote repository URL for all git projects in the current directory.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		newRemote, err := cmd.Flags().GetString("new-remote")
		if err != nil {
			return fmt.Errorf("failed to get new-remote flag: %w", err)
		}

		if newRemote == "" {
			return fmt.Errorf("new-remote flag is required")
		}

		currDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		gs := service.NewGitService(mrRepoLogger)

		entries, err := os.ReadDir(currDir)
		if err != nil {
			return fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			absPath := filepath.Join(currDir, entry.Name())

			if err := gs.UpdateRemote(context.Background(), absPath, newRemote); err != nil {
				mrRepoLogger.Warn("UpdateRemote: ", absPath, err.Error())
			}
		}
		return nil
	},
}

func init() {
	updateRemoteCmd.Flags().StringP("new-remote", "a", "", "New remote URL (required)")
}
