package main

import (
	"fmt"

	cobra "github.com/spf13/cobra"
	"github.com/untillpro/qg/cmdupload"
	"github.com/untillpro/qg/vcs"
)

func main() {

	var rootCmd = &cobra.Command{
		Use:   "qg",
		Short: "Quick git wrapper",
		Run: func(cmd *cobra.Command, args []string) {
			vcs.Status()
		},
	}
	var downloadCmd = &cobra.Command{
		Use:   "d",
		Short: "Download sources",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Inside download with args: %v\n", args)
		},
	}

	rootCmd.AddCommand(cmdupload.Command())
	rootCmd.AddCommand(downloadCmd)

	rootCmd.Execute()

}
