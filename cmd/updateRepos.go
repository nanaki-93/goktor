/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/nanaki-93/goktor/service"

	"github.com/spf13/cobra"
)

// folderListCmd represents the updateRepos command
var updateReposCmd = &cobra.Command{
	Use:   "updateRepos",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		fs := service.NewService()

		res, err := fs.ListDirectories("C:/")
		if err != nil {
			fmt.Println("error:", err)
		}
		dirs := res.FlattenDirectory()
		fmt.Println(dirs)

	},
}

func init() {
	rootCmd.AddCommand(updateReposCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// updateRepostCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// updateReposCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
