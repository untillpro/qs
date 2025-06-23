package commands

import (
	"context"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	contextPkg "github.com/untillpro/qs/internal/context"
	"strings"

	"github.com/untillpro/qs/gitcmds"
	"github.com/untillpro/qs/internal/types"
	"github.com/untillpro/qs/vcs"
)

func U(cmd *cobra.Command, cfgUpload vcs.CfgUpload, wd string) error {
	globalConfig()
	if err := gitcmds.Status(wd); err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}

	files := gitcmds.GetFilesForCommit(wd)
	if len(files) == 0 {
		return errors.New("There is nothing to commit")
	}

	// find out type of the branch
	branchType := gitcmds.GetBranchType(wd)
	// if branch type is unknown, we cannot proceed
	if branchType == types.BranchTypeUnknown {
		return errors.New("You must be on either a pr or dev branch")
	}

	// calculate total length of commit message parts
	totalLength := 0
	if len(cfgUpload.Message) > 0 {
		totalLength = len(strings.Join(cfgUpload.Message, " "))
	}

	// each branch type has different tolerance to the length of the commit message
	finalCommitMessages := make([]string, 0, len(cfgUpload.Message))
	switch branchType {
	case types.BranchTypeDev:
		if totalLength == 0 {
			// for dev branch default commit message is "dev"
			finalCommitMessages = append(finalCommitMessages, gitcmds.PushDefaultMsg)
		} else {
			finalCommitMessages = append(finalCommitMessages, cfgUpload.Message...)
		}
	case types.BranchTypePr:
		// if commit message is not specified or is shorter than 8 characters
		if totalLength < 8 {
			return errors.New("Commit message is missing or too short (minimum 8 characters)")
		}

		finalCommitMessages = append(finalCommitMessages, cfgUpload.Message...)
	}
	// put commit message to context
	cmd.SetContext(context.WithValue(cmd.Context(), contextPkg.CtxKeyCommitMessage, strings.Join(finalCommitMessages, " ")))

	return gitcmds.Upload(wd, finalCommitMessages)
}
