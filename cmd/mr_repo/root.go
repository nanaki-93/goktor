/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package mr_repo

import (
	"github.com/nanaki-93/goktor/cmd"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mr-repo",
	Short: "Manage multiple repositories",
	Long:  `Commands to manage multiple git repositories in a directory.`,
}

func init() {
	cmd.RootCmd.AddCommand(rootCmd)
}
