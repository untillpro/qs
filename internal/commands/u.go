package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/gitcmds"
	notesPkg "github.com/untillpro/qs/internal/notes"
	"github.com/untillpro/qs/utils"
)

func U(cmd *cobra.Command, commitMessage string, wd string) error {
	currentBranch, _, isMain, err := gitcmds.GetCurrentBranchInfo(wd)
	if err != nil {
		return err
	}
	if isMain {
		logger.Verbose("You are in main branch.")
	}

	// Fetch notes from origin
	_, _, err = new(exec.PipedExec).
		Command("git", "fetch", "origin", "--force", utils.RefsNotes).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(fmt.Sprintf("Failed to fetch notes: %v", err))
		// Continue anyway, as notes might exist locally
	}

	files := gitcmds.GetFilesForCommit(wd)
	neetToCommit := len(files) > 0
	// If there are files to commit, set commit message
	if neetToCommit {
		if err := setCommitMessage(cmd, commitMessage, wd, isMain); err != nil {
			return err
		}

		// Ensure large file hook content is up to date
		if err := gitcmds.EnsureLargeFileHookUpToDate(wd); err != nil {
			logger.Verbose("Error updating large file hook content:", err)
		}
	}

	return gitcmds.Upload(cmd, wd, currentBranch, neetToCommit)
}

func setCommitMessage(
	cmd *cobra.Command,
	commitMessage string,
	wd string,
	isMainBranch bool,
) error {
	_, branchType, err := gitcmds.GetBranchType(wd)
	if err != nil {
		return err
	}

	switch branchType {
	case notesPkg.BranchTypeDev:
		if commitMessage == "" {
			commitMessage = gitcmds.DefaultCommitMessage
		}
	case notesPkg.BranchTypePr:
		switch {
		case commitMessage == "":
			return ErrEmptyCommitMessage
		case len(commitMessage) < minimumCommitMessageLen:
			return ErrShortCommitMessage
		}
	default:
		if commitMessage == "" {
			if isMainBranch {
				return ErrEmptyCommitMessage
			}
			commitMessage = gitcmds.DefaultCommitMessage
		}
	}

	cmd.SetContext(context.WithValue(cmd.Context(), utils.CtxKeyCommitMessage, commitMessage))

	return nil
}
