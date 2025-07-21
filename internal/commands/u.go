package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

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

	_, isMain, err := gitcmds.IamInMainBranch(wd)
	if err != nil {
		return err
	}
	if isMain {
		fmt.Println("You are in main branch.")
	}

	// Fetch notes from origin
	_, _, err = new(exec.PipedExec).
		Command("git", "fetch", "origin", "--force", "refs/notes/*:refs/notes/*").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose("Failed to fetch notes: %v", err)
		// Continue anyway, as notes might exist locally
	}

	files := gitcmds.GetFilesForCommit(wd)
	if len(files) == 0 {
		return errors.New("there is nothing to commit")
	}

	if err := setCommitMessage(cmd, cfgUpload, wd); err != nil {
		return err
	}

	return gitcmds.Upload(cmd, wd)
}

func setCommitMessage(cmd *cobra.Command, cfgUpload vcs.CfgUpload, wd string) error {
	defaultCommitMessage := os.Getenv(EnvMainCommitMessage)
	// find out a type of the branch
	branchType, err := gitcmds.GetBranchType(wd)
	if err != nil {
		return err
	}

	// calculate total length of commit message parts
	totalLength := 0
	if len(cfgUpload.Message) > 0 {
		totalLength = len(strings.Join(cfgUpload.Message, " "))
	}

	// each branch type has different tolerance to the length of the commit message
	finalCommitMessages := make([]string, 0, len(cfgUpload.Message))
	switch branchType {
	case notesPkg.BranchTypeDev:
		if totalLength == 0 {
			// for dev branch default commit message is "dev"
			finalCommitMessages = append(finalCommitMessages, gitcmds.PushDefaultMsg)
		} else {
			finalCommitMessages = append(finalCommitMessages, cfgUpload.Message...)
		}
	case notesPkg.BranchTypePr:
		// if a commit message is not specified or is shorter than 8 characters
		if totalLength < 8 {
			return errors.New("commit message is missing or too short (minimum 8 characters)")
		}

		finalCommitMessages = append(finalCommitMessages, cfgUpload.Message...)
	default:
		if totalLength == 0 {
			if len(strings.TrimSpace(defaultCommitMessage)) > 0 {
				finalCommitMessages = append(finalCommitMessages, defaultCommitMessage)
			} else {
				commitMessageParts, err := readCommitMessage()
				if err != nil {
					return err
				}
				finalCommitMessages = append(finalCommitMessages, commitMessageParts...)
			}
		}
	}
	// put commit a message to context
	cmd.SetContext(context.WithValue(cmd.Context(), contextPkg.CtxKeyCommitMessage, finalCommitMessages))

	return nil
}

// readCommitMessage from stdin
func readCommitMessage() ([]string, error) {
	fmt.Print("Enter commit message (press Enter to confirm): ")
	reader := bufio.NewReader(os.Stdin)
	commitMessage, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	commitMessageParts := strings.Fields(commitMessage)
	if len(commitMessageParts) == 0 {
		return nil, errors.New("commit message is empty")
	}

	return commitMessageParts, nil
}
