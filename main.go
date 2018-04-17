package main

import (
	"fmt"

	cobra "github.com/spf13/cobra"
	v "github.com/untillpro/qg/vcs"
)

func main() {

	var rootCmd = &cobra.Command{
		Use:   "qg",
		Short: "Quick git wrapper",
		Run: func(cmd *cobra.Command, args []string) {
			v.Status()
		},
	}

	var uploadCmdMessage []string
	var uploadCmd = &cobra.Command{
		Use:   "u",
		Short: "Upload sources",
		Run: func(cmd *cobra.Command, args []string) {
			v.Upload(uploadCmdMessage)
		},
	}
	uploadCmd.Flags().StringSliceVarP(&uploadCmdMessage, "message", "m", []string{"misc"},
		`Use the given as the commit message. If multiple -m options are given
their values are concatenated as separate paragraphs`,
	)

	var downloadCmd = &cobra.Command{
		Use:   "d",
		Short: "Download sources",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Inside download with args: %v\n", args)
		},
	}

	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(downloadCmd)

	rootCmd.Execute()

}
