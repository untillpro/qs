/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package gitcmds

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/utils"
)

// Upload uploads sources to git repo
func Upload(cmd *cobra.Command, wd, currentBranch string, needToCommit bool) error {
	if needToCommit {
		commitMessage := cmd.Context().Value(utils.CtxKeyCommitMessage).(string)

		stdout, stderr, err := new(exec.PipedExec).
			Command(git, "add", ".").
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("git add failed: %w", err)
		}
		logger.Verbose(stdout)

		params := []string{"commit", "-a", mimm, commitMessage}

		_, stderr, err = new(exec.PipedExec).
			Command(git, params...).
			WorkingDir(wd).
			RunToStrings()
		if strings.Contains(stderr, MsgPreCommitError) {
			var response string
			fmt.Println("")
			printLn(strings.TrimSpace(stderr))
			fmt.Print("Do you want to commit anyway(y/n)?")
			_, _ = fmt.Scanln(&response)

			if response != "y" {
				return nil
			}

			params = append(params, "-n")
			_, stderr, err = new(exec.PipedExec).
				Command(git, params...).
				WorkingDir(wd).
				RunToStrings()
		}
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("git commit failed: %w", err)
		}
	}

	// make pull before push
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, pull).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("error pulling before push: %w", err)
	}

	err = utils.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, push, origin, utils.RefsNotes).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("git push notes failed: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	utils.DelayIfTest()

	hasUpstream, err := hasUpstreamBranch(wd, currentBranch)
	if err != nil {
		return fmt.Errorf("failed to check upstream branch: %w", err)
	}

	pushArgs := []string{push, origin, currentBranch}
	if !hasUpstream {
		pushArgs = []string{push, "-u", origin, currentBranch}
	}

	err = utils.Retry(func() error {
		var pushErr error
		stdout, stderr, pushErr = new(exec.PipedExec).
			Command(git, pushArgs...).
			WorkingDir(wd).
			RunToStrings()
		if pushErr != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("git push failed: %w", pushErr)
		}

		return nil
	})
	if err != nil {
		return err
	}
	logger.Verbose(stdout)

	return nil
}

// hasUpstreamBranch checks if the current branch has an upstream tracking branch configured
func hasUpstreamBranch(wd string, branchName string) (bool, error) {
	stdout, _, err := new(exec.PipedExec).
		Command(git, "config", "--get", fmt.Sprintf("branch.%s.remote", branchName)).
		WorkingDir(wd).
		RunToStrings()

	if err != nil {
		// If the config doesn't exist, git config returns exit code 1
		// This is expected when no upstream is configured
		return false, nil
	}

	return strings.TrimSpace(stdout) != "", nil
}
