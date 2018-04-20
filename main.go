package main

import (
	"io/ioutil"
	"log"

	cobra "github.com/spf13/cobra"
	"github.com/untillpro/qg/git"
	"github.com/untillpro/qg/vcs"
)

var verbose bool

func globalConfig() {
	if !verbose {
		log.SetOutput(ioutil.Discard)
	}
}

func main() {

	var cfgStatus vcs.CfgStatus
	var rootCmd = &cobra.Command{
		Use:   "qg",
		Short: "Quick git wrapper",
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig()
			git.Status(cfgStatus)
		},
	}
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Upload
	{
		var cfgUpload vcs.CfgUpload
		var uploadCmd = &cobra.Command{
			Use:   "u",
			Short: "Upload sources to repo",
			Run: func(cmd *cobra.Command, args []string) {
				globalConfig()
				git.Upload(cfgUpload)
			},
		}

		uploadCmd.Flags().StringSliceVarP(&cfgUpload.Message, "message", "m", []string{"misc"},
			`Use the given string as the commit message. If multiple -m options are given
their values are concatenated as separate paragraphs`,
		)
		rootCmd.AddCommand(uploadCmd)
	}

	// Download
	{
		var cfg vcs.CfgDownload
		var cmd = &cobra.Command{
			Use:   "d",
			Short: "Download sources from repo",
			Run: func(cmd *cobra.Command, args []string) {
				globalConfig()
				git.Download(cfg)
			},
		}
		rootCmd.AddCommand(cmd)
	}
	rootCmd.Execute()

}
