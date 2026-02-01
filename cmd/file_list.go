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

// fileListCmd represents the fileList command
var fileListCmd = &cobra.Command{
	Use:   "file-list",
	Short: "List files and their sizes",
	Long:  `List all files recursively with their sizes in the specified directory.`,
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
		res, err := fs.ListFiles(dirToScan)
		if err != nil {
			return fmt.Errorf("failed to list files: %w", err)
		}

		fs.PrintFiles(res)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(fileListCmd)
	fileListCmd.Flags().StringP("dir", "d", "", "Directory to scan (defaults to current directory)")

}
