package cmd

import (
	"fmt"

	"github.com/nanaki-93/goktor/model"
	"github.com/nanaki-93/goktor/service"
	"github.com/spf13/cobra"
)

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Get the diff from 2 input files",
	Long: `Receive in input 2 csv files with 2 column, a key and a content.
The diff compare the 2 content between the 2 files searching with the key.
The output will be a csv file with 2 column, a key and a result Yes for equal content, No for different content.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		leftFile, err := cmd.Flags().GetString("left")
		if err != nil {
			return fmt.Errorf("failed to get left flag: %w", err)
		}

		rightFile, err := cmd.Flags().GetString("right")
		if err != nil {
			return fmt.Errorf("failed to get right flag: %w", err)
		}

		delimiter, err := cmd.Flags().GetString("delimiter")
		if err != nil {
			return fmt.Errorf("failed to get delimiter flag: %w", err)
		}

		output, err := cmd.Flags().GetBool("output")
		if err != nil {
			return fmt.Errorf("failed to get output flag: %w", err)
		}

		hasHeader, err := cmd.Flags().GetBool("header")
		if err != nil {
			return fmt.Errorf("failed to get header flag: %w", err)
		}

		diffServ := service.NewDiffService()
		if err != nil {
			return fmt.Errorf("failed to create diff service: %w", err)
		}

		resultPath, err := diffServ.CalculateDiff(model.DiffConfig{
			Left:       leftFile,
			Right:      rightFile,
			Delimiter:  rune(delimiter[0]),
			WithHeader: hasHeader,
			Output:     output})
		if err != nil {
			return fmt.Errorf("failed to run diff service: %w", err)
		}

		fmt.Println("Your result file is ", resultPath)
		return nil
	},
}

func init() {
	diffCmd.Flags().StringP("left", "l", "", "left file to compare")
	diffCmd.Flags().StringP("right", "r", "", "right file to compare")
	diffCmd.Flags().StringP("delimiter", "d", "\t", "delimiter for columns")
	diffCmd.Flags().BoolP("output", "o", false, "write all the detail for the diff")
	diffCmd.Flags().BoolP("header", "H", false, "the input has header or not")
}
