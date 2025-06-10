package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/untillpro/qs/git"
	"github.com/untillpro/qs/internal/types"
	"github.com/untillpro/qs/vcs"
)

func U(cfgUpload vcs.CfgUpload) error {
	var response string

	globalConfig()
	if err := git.Status(); err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}

	files := git.GetFilesForCommit()
	if len(files) == 0 {
		return errors.New("There is nothing to commit")
	}

	// find out type of the branch
	branchType := git.GetBranchType()
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
			finalCommitMessages = append(finalCommitMessages, git.PushDefaultMsg)
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

	// ask for confirmation before pushing
	pushConfirm := pushConfirm + " with comment: \n\n'" + strings.Join(finalCommitMessages, " ") + "'\n\n'y': agree, 'g': show GUI >"
	fmt.Print(pushConfirm)
	_, _ = fmt.Scanln(&response)
	// handle user response
	switch response {
	case pushYes:
		return git.Upload(finalCommitMessages)
	case guiParam:
		return git.Gui()
	default:
		fmt.Print(pushFail)

		return nil
	}
}
