package cmdupload

import (
	cobra "github.com/spf13/cobra"
	"github.com/untillpro/qg/vcs"
)

var message []string

// Command creates a command
func Command() *cobra.Command {
	var uploadCmd = &cobra.Command{
		Use:   "u",
		Short: "Upload sources",
		Run: func(cmd *cobra.Command, args []string) {
			vcs.Upload(message)
		},
	}

	uploadCmd.Flags().StringSliceVarP(&message, "message", "m", []string{"misc"},
		`Use the given as the commit message. If multiple -m options are given
	their values are concatenated as separate paragraphs`,
	)
	return uploadCmd
}
