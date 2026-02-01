/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

var mrRepo = &cobra.Command{
	Use:   "mr-repo",
	Short: "Manage multiple repositories",
	Long:  `Commands to manage multiple git repositories in a directory.`,
}

func init() {
	rootCmd.AddCommand(mrRepo)
}
