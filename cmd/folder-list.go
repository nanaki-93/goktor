/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/nanaki-93/goktor/service"

	"github.com/spf13/cobra"
)

// folderListCmd represents the folderList command
var folderListCmd = &cobra.Command{
	Use:   "folderList",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		fs := service.NewService()

		res, err := fs.ListDirectories("D:/")
		if err != nil {
			fmt.Println("error:", err)
		}

		fs.PrintDirectories(service.ReorderDirectory(res), fs.GetSizeFilter())
	},
}

func init() {
	rootCmd.AddCommand(folderListCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// folderListCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// folderListCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
