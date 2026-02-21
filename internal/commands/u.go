package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/gitcmds"
	contextPkg "github.com/untillpro/qs/internal/context"
	notesPkg "github.com/untillpro/qs/internal/notes"
	"github.com/untillpro/qs/vcs"
)

func U(cmd *cobra.Command, cfgUpload vcs.CfgUpload, wd string) error {
	if err := gitcmds.Status(wd); err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}

	currentBranch, isMain, err := gitcmds.IamInMainBranch(wd)
	if err != nil {
		return err
	}
	if isMain {
		logger.Verbose("You are in main branch.")
	}

	// Fetch notes from origin
	_, _, err = new(exec.PipedExec).
		Command(git, fetch, origin, "--force", refsNotes).
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
		if err := setCommitMessage(cmd, cfgUpload, wd, isMain); err != nil {
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
	cfgUpload vcs.CfgUpload,
	wd string,
	isMainBranch bool,
) error {
	_, branchType, err := gitcmds.GetBranchType(wd)
	if err != nil {
		return err
	}

	msg := cfgUpload.Message
	switch branchType {
	case notesPkg.BranchTypeDev:
		if msg == "" {
			msg = gitcmds.DefaultCommitMessage
		}
	case notesPkg.BranchTypePr:
		switch {
		case msg == "":
			return ErrEmptyCommitMessage
		case len(msg) < minimumCommitMessageLen:
			return ErrShortCommitMessage
		}
	default:
		if msg == "" {
			if isMainBranch {
				return ErrEmptyCommitMessage
			}
			msg = gitcmds.DefaultCommitMessage
		}
	}

	cmd.SetContext(context.WithValue(cmd.Context(), contextPkg.CtxKeyCommitMessage, msg))

	return nil
}
