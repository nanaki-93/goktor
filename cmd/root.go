// Package cmd
/*
	Copyright © 2025 Marco Andreose <andreose.marco93@gmail.com>
*/

package cmd

import (
	"fmt"
	"os"

	"github.com/nanaki-93/goktor/cmd/mr_repo"
	"github.com/nanaki-93/goktor/service"
	"github.com/spf13/cobra"
)

var GlobalLogger service.Logger

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "goktor",
	Short: "A CLI tool for managing directories and repositories",
	Long: `Goktor is a command-line utility for analyzing directory structures,
listing files and their sizes, and managing multiple git repositories.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		debug, _ := cmd.Flags().GetBool("verbose")
		GlobalLogger = service.NewLogger(debug)
		mr_repo.SetLogger(GlobalLogger)
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		GlobalLogger.Error("Failed to execute command: \n", err, "\n")
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose output")
	RootCmd.CompletionOptions.DisableDefaultCmd = false

	// Add subcommands here
	RootCmd.AddCommand(fileListCmd)
	RootCmd.AddCommand(folderListCmd)
	RootCmd.AddCommand(mr_repo.MrRepoCmd)
}
