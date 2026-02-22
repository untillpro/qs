/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package gitcmds

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/utils"
)

// Fork repo
func Fork(wd string) (string, error) {
	repo, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return "", err
	}

	if len(repo) == 0 {
		return "", errors.New(repoNotFound)
	}

	remoteURL := GetRemoteUpstreamURL(wd)
	if len(remoteURL) > 0 {
		return repo, errors.New(ErrAlreadyForkedMsg)
	}

	if ok, err := IsMainOrg(wd); !ok || err != nil {
		if err != nil {
			return repo, fmt.Errorf("IsMainOrg error: %w", err)
		}

		return repo, errors.New(ErrAlreadyForkedMsg)
	}

	_, chExist, err := ChangedFilesExist(wd)
	if err != nil {
		return "", err
	}
	if chExist {
		stdout, stderr, err := new(exec.PipedExec).
			Command(git, "add", ".").
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return repo, errors.New(stderr)
			}

			return repo, fmt.Errorf("git add failed: %w", err)
		}
		printLn(stdout)

		stdout, stderr, err = new(exec.PipedExec).
			Command(git, "stash").
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return repo, errors.New(stderr)
			}

			return repo, fmt.Errorf("git stash failed: %w", err)
		}
		printLn(stdout)
	}

	var (
		stdout string
		stderr string
	)
	err = utils.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command("gh", "repo", "fork", org+slash+repo, "--clone=false").
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("failed to fork repository: %w", err)
		}

		return nil
	})
	if err != nil {
		return repo, err
	}
	printLn(stdout)

	// Get current user name to verify fork
	userName, err := getUserName(wd)
	if err != nil {
		logger.Verbose(fmt.Sprintf("Failed to get user name for verification: %v", err))

		return repo, err
	}

	// Verify fork was created and is accessible with retry
	err = utils.Retry(func() error {
		// Try to get user email to get a valid token context, then verify repo
		userEmail, emailErr := utils.GetUserEmail()
		if emailErr != nil {
			return fmt.Errorf("failed to verify GitHub authentication: %w", emailErr)
		}
		logger.Verbose(fmt.Sprintf("Verified GitHub authentication for user: %s", userEmail))

		// Verify the forked repository exists and is accessible
		return utils.VerifyGitHubRepoExists(userName, repo, "")
	})
	if err != nil {
		logger.Verbose(fmt.Sprintf("Fork verification failed: %v", err))

		return repo, fmt.Errorf("fork verification failed: %w", err)
	}
	_, _ = fmt.Fprintln(os.Stdout, "Fork created and verified successfully")

	return repo, nil
}

func GetRemoteUpstreamURL(wd string) string {
	stdout, _, err := new(exec.PipedExec).
		Command(git, "config", "--local", "remote.upstream.url").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(stdout)
}

func IsMainOrg(wd string) (bool, error) {
	_, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return false, err
	}
	userName, err := getUserName(wd)

	return org != userName, err
}

// MakeUpstream s.e.
func MakeUpstream(wd string, repo string) error {
	userName, err := getUserName(wd)
	if err != nil {
		return fmt.Errorf("failed to get user name: %w", err)
	}

	if len(userName) == 0 {
		return errors.New(userNotFound)
	}

	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "remote", "rename", "origin", "upstream").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to rename origin to upstream: %w", err)
	}
	printLn(stdout)

	stdout, stderr, err = new(exec.PipedExec).
		Command(git, "remote", "add", "origin", "https://github.com/"+userName+slash+repo).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to add origin remote: %w", err)
	}
	printLn(stdout)

	// delay to ensure remote is added
	utils.DelayIfTest()

	err = utils.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, "fetch", "origin").
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("failed to fetch origin: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}
	logger.Verbose(stdout)

	stdout, stderr, err = new(exec.PipedExec).
		Command(git, branch, "--set-upstream-to", originSlash+mainBranch, mainBranch).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to set upstream for main branch: %w", err)
	}
	printLn(stdout)

	return nil
}

func getUserName(wd string) (string, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command("gh", "api", "user").
		WorkingDir(wd).
		Command("jq", "-r", ".login").
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", errors.New(stderr)
		}

		return "", fmt.Errorf("failed to get user name: %w", err)
	}

	return strings.TrimSpace(stdout), nil
}
