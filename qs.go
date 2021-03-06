package main

import (
	"fmt"

	cobra "github.com/spf13/cobra"
	qc "github.com/untillpro/gochips"
	"github.com/untillpro/qs/git"
	"github.com/untillpro/qs/vcs"
)

var verbose bool

func globalConfig() {
	qc.IsVerbose = verbose

}

func main() {

	var cfgStatus vcs.CfgStatus
	var rootCmd = &cobra.Command{
		Use:   "qs",
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
			Use: "u",
			//			Aliases: []string{"u"},
			Short: "Upload sources to repo",
			Run: func(cmd *cobra.Command, args []string) {
				globalConfig()
				git.Status(cfgStatus)
				fmt.Print("\n*** Changes shown above will be uploaded to repository, 'y': agree, 'g': show GUI >")
				var response string
				fmt.Scanln(&response)
				switch response {
				case "y":
					git.Upload(cfgUpload)
				case "g":
					git.Gui()
				default:
					fmt.Print("Ok, see you")
				}
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

	// Release
	{
		var cmd = &cobra.Command{
			Use:   "r",
			Short: "Create a release",
			Run: func(cmd *cobra.Command, args []string) {
				globalConfig()
				git.Release()
			},
		}
		rootCmd.AddCommand(cmd)
	}

	// GUI
	{
		var cmd = &cobra.Command{
			Use:   "g",
			Short: "Show GUI",
			Run: func(cmd *cobra.Command, args []string) {
				globalConfig()
				git.Gui()
			},
		}
		rootCmd.AddCommand(cmd)
	}

	rootCmd.Execute()

}
