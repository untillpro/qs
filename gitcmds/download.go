/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package gitcmds

import (
	"errors"
	"fmt"
	"strings"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/utils"
)

// Download sources from git repo
func Download(wd string) error {
	// Step 1: Exit if there are uncommitted changes
	uncommittedChanges, err := HaveUncommittedChanges(wd)
	if err != nil {
		return err
	}

	if uncommittedChanges {
		return errors.New("there are uncommitted changes in the repository")
	}

	var (
		stderr string
		stdout string
	)
	// Step 2: fetch origin --prune
	err = utils.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, fetch, origin, "--prune").
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("failed to fetch origin --prune: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}
	logger.Verbose(stdout)

	// Step 3: git fetch origin --force refs/notes/*:refs/notes/*
	err = utils.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, fetch, origin, "--force", utils.RefsNotes).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("failed to fetch notes: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}
	logger.Verbose(stdout)

	currentBranchName, mainBranchName, isMain, err := GetCurrentBranchInfo(wd)
	if err != nil {
		return err
	}

	// check out on the main branch
	if !isMain {
		if err := CheckoutOnBranch(wd, mainBranchName); err != nil {
			return err
		}
	}

	// Step 4: merge origin Main => Main with fast-forward only
	_, stderr, err = new(exec.PipedExec).
		Command(git, "merge", "--ff-only", fmt.Sprintf("origin/%s", mainBranchName)).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		// Check if fast-forward failed
		if checkAndShowFastForwardFailure(stderr, mainBranchName) {
			return fmt.Errorf("cannot fast-forward merge origin/%s", mainBranchName)
		}

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to merge origin/%s with --ff-only: %w", mainBranchName, err)
	}

	// check out back on the previous branch
	if !isMain {
		if err := CheckoutOnBranch(wd, currentBranchName); err != nil {
			return err
		}
	}

	// Step 5: If not on Main and the remote tracking branch exists merge local branch with the remote branch
	if !isMain {
		var hasRemoteBranch bool

		hasRemoteBranch, err = hasRemoteTrackingBranch(wd, currentBranchName)
		if err != nil {
			return fmt.Errorf("failed to check remote tracking branch: %w", err)
		}

		if hasRemoteBranch {
			_, stderr, err = new(exec.PipedExec).
				Command(git, "merge", fmt.Sprintf("origin/%s", currentBranchName)).
				WorkingDir(wd).
				RunToStrings()
			if err != nil {
				logger.Verbose(stderr)

				if len(stderr) > 0 {
					return errors.New(stderr)
				}

				return fmt.Errorf("failed to merge origin/%s: %w", currentBranchName, err)
			}
		}
	}

	// Step 6: If upstream exists - pull upstream/Main with fast-forward only
	upstreamExists, err := HasRemote(wd, "upstream")
	if err != nil {
		return fmt.Errorf("failed to check if upstream exists: %w", err)
	}

	if upstreamExists {
		if !isMain {
			if err := CheckoutOnBranch(wd, mainBranchName); err != nil {
				return err
			}
		}

		err = utils.Retry(func() error {
			stdout, stderr, err = new(exec.PipedExec).
				Command(git, pull, "--ff-only", "upstream", mainBranchName).
				WorkingDir(wd).
				RunToStrings()
			if err != nil {
				logger.Verbose(stderr)
				if checkAndShowFastForwardFailure(stderr, mainBranchName) {
					return fmt.Errorf("cannot fast-forward merge upstream/%s", mainBranchName)
				}
				if len(stderr) > 0 {
					return errors.New(stderr)
				}

				return fmt.Errorf("failed to pull upstream/%s with --ff-only: %w", mainBranchName, err)
			}

			return nil
		})
		if err != nil {
			return err
		}
		logger.Verbose(stdout)

		if !isMain {
			if err := CheckoutOnBranch(wd, currentBranchName); err != nil {
				return err
			}
		}
	}

	return nil
}

// hasRemoteTrackingBranch checks if a remote tracking branch exists for the given branch
func hasRemoteTrackingBranch(wd string, branchName string) (bool, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "branch", "-r").
		WorkingDir(wd).
		Command("grep", branchName).
		RunToStrings()
	if len(stdout) == 0 {
		return false, nil
	}

	if err != nil {
		logger.Verbose(stderr)

		return false, err
	}

	strBranches := strings.TrimSpace(stdout)
	if strBranches == "" {
		return false, nil
	}

	rBranches := strings.Split(strBranches, "\n")
	for _, rBranch := range rBranches {
		if strings.TrimSpace(rBranch) == "origin/"+branchName {
			return true, nil
		}
	}

	return false, nil
}
