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

// folderListCmd represents the updateRepos command
var mrRepo = &cobra.Command{
	Use:   "mr-repo",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

var updateRemoteCmd = &cobra.Command{
	Use:   "update-remote",
	Short: "A brief description of your command",
	Long:  `Update the remote repositories for all git projects in the specified directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("update-remote called")

		newRemoteValue := cmd.Flag("new-remote").Value.String()
		fmt.Println("new remote value:", newRemoteValue)
		currDir, err := os.Getwd()
		if err != nil {
			fmt.Println("error getting current directory:", err)
			return
		}
		gs := service.NewGitService()

		entries, err := os.ReadDir(currDir)

		for _, entry := range entries {
			if entry.IsDir() {
				fmt.Println("processing dir:", entry.Name())
				absPath := currDir + string(os.PathSeparator) + entry.Name()

				err = gs.UpdateRemote(absPath, newRemoteValue)
				if err != nil {
					fmt.Println("error updating remote for the repository:", absPath, err)
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(mrRepo)
	mrRepo.AddCommand(updateRemoteCmd)
	mrRepo.PersistentFlags().StringP("new-remote", "a", "", "Add a new remote to all repositories")

}
