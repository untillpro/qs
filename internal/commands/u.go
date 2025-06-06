package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/untillpro/qs/git"
	"github.com/untillpro/qs/internal/types"
	"github.com/untillpro/qs/vcs"
)

func U(cfgStatus vcs.CfgStatus, cfgUpload vcs.CfgUpload, args []string) {
	var response string

	globalConfig()
	git.Status(cfgStatus)

	files := git.GetFilesForCommit()
	if len(files) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "There is nothing to commit")
		os.Exit(1)

		return
	}

	// find out type of the branch
	branchType, err := git.GetBranchType()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error getting branch type: %v", err)
		os.Exit(1)

		return
	}

	// if branch type is unknown, we cannot proceed
	if branchType == types.BranchTypeUnknown {
		_, _ = fmt.Fprintln(os.Stderr, "You must be on either a pr or dev branch")
		os.Exit(1)

		return
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
			finalCommitMessages = append(finalCommitMessages, git.PushDefaultMsg)
		} else {
			finalCommitMessages = append(finalCommitMessages, cfgUpload.Message...)
		}
	case types.BranchTypePr:
		// if commit message is not specified or is shorter than 8 characters
		if totalLength < 8 {
			_, _ = fmt.Fprintln(os.Stderr, "Commit message is missing or too short (minimum 8 characters)")
			os.Exit(1)

			return
		}

		finalCommitMessages = append(finalCommitMessages, cfgUpload.Message...)
	}

	// ask for confirmation before pushing
	pushConfirm := pushConfirm + " with comment: \n\n'" + strings.Join(finalCommitMessages, " ") + "'\n\n'y': agree, 'g': show GUI >"
	fmt.Print(pushConfirm)
	_, _ = fmt.Scanln(&response)
	// handle user response
	switch response {
	case pushYes:
		git.Upload(finalCommitMessages)
	case guiParam:
		git.Gui()
	default:
		fmt.Print(pushFail)
	}
}
