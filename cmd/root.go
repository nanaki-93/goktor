// Package cmd
/*
	Copyright © 2025 Marco Andreose <andreose.marco93@gmail.com>
*/

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "goktor",
	Short: "A CLI tool for managing directories and repositories",
	Long: `Goktor is a command-line utility for analyzing directory structures,
listing files and their sizes, and managing multiple git repositories.`,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.CompletionOptions.DisableDefaultCmd = false
}
