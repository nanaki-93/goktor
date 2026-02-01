/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/nanaki-93/goktor/service"

	"github.com/spf13/cobra"
)

// folderListCmd represents the folderList command
var folderListCmd = &cobra.Command{
	Use:   "folder-list",
	Short: "List directories and their sizes",
	Long: `List all directories recursively with their total sizes.
You can specify a directory to scan or use the current directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		dirToScan, err := cmd.Flags().GetString("dir")
		if err != nil {
			return fmt.Errorf("failed to get dir flag: %w", err)
		}

		if dirToScan == "" {
			var err error
			dirToScan, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
		}

		fs := service.NewService()

		res, err := fs.ListDirectories(dirToScan)
		if err != nil {
			return fmt.Errorf("failed to list directories: %w", err)
		}

		fs.PrintDirectories(service.ReorderDirectory(res), fs.GetSizeFilter())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(folderListCmd)
	folderListCmd.Flags().StringP("dir", "d", "", "Directory to scan (defaults to current directory)")
}
