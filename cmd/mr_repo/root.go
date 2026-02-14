/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package mr_repo

import (
	"github.com/nanaki-93/goktor/service"
	"github.com/spf13/cobra"
)

var mrRepoLogger service.Logger

func SetLogger(logger service.Logger) {
	mrRepoLogger = logger
}

var MrRepoCmd = &cobra.Command{
	Use:   "mr-repo",
	Short: "Manage multiple repositories",
	Long:  `Commands to manage multiple git repositories in a directory.`,
}

func init() {
	MrRepoCmd.AddCommand(updateRemoteCmd)
}
