package mr_repo

import (
	"context"
	"fmt"
	"os"

	"github.com/nanaki-93/goktor/service"
	"github.com/spf13/cobra"
)

var deleteMergedCmd = &cobra.Command{
	Use:          "delete-merged",
	Short:        "Delete all the merged branches with a end date",
	Long:         `Delete all the merged branches with a end date passed as a mandatory argument, with format YYYY-MM-DD`,
	SilenceUsage: true,
	Args:         cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		endDate := args[0]
		if endDate == "" {
			return fmt.Errorf("a new remote arg is required")
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		currDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		gs := service.NewGitService(mrRepoLogger)

		deletedBranches, err := gs.DeleteMergedBranches(ctx, currDir, endDate, dryRun)
		if err != nil {
			return fmt.Errorf("failed to Delete merged branches: %w", err)
		}

		for _, result := range deletedBranches {
			logResult(result, dryRun)
		}

		return nil
	},
}

func logResult(result service.DeleteMergedBranchesResult, dryRun bool) {
	if dryRun {
		for _, branch := range result.DryRun {
			mrRepoLogger.Info("DryRun branch to delete:", branch)
		}
		return
	}

	for _, branch := range result.Deleted {
		mrRepoLogger.Info("Removed branch: ", branch)
	}
	for _, branch := range result.Skipped {
		mrRepoLogger.Info("Skipped branch: ", branch)
	}
	for _, branch := range result.Failed {
		mrRepoLogger.Warn("Failed branch: ", branch)
	}
}

func init() {

	deleteMergedCmd.Flags().BoolP("dry-run", "d", false, "dry run")
}
