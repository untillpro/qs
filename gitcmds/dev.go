/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package gitcmds

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/utils"
)

// CreateDevBranch creates dev branch and pushes it to origin
func CreateDevBranch(wd, branchName, mainBranch string, notes []string) error {
	branchName = normalizeBranchName(branchName)
	if branchName == "" {
		return errors.New("branch name is empty after normalization")
	}

	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "checkout", mainBranch).
		WorkingDir(wd).
		RunToStrings()

	if err != nil {
		if strings.Contains(err.Error(), err128) && strings.Contains(stderr, "matched multiple") {
			err = new(exec.PipedExec).
				Command(git, "checkout", "--track", originSlash+mainBranch).
				WorkingDir(wd).
				Run(os.Stdout, os.Stdout)
		}
	}
	if err != nil {
		return err
	}

	// Create new branch from main
	err = new(exec.PipedExec).
		Command(git, "checkout", "-B", branchName).
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return err
	}

	// Fetch notes from origin before pushing
	stdout, stderr, err = new(exec.PipedExec).
		Command(git, fetch, origin, "--force", utils.RefsNotes).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		return fmt.Errorf("failed to fetch notes: %w, stdout: %s", err, stdout)
	}

	// Add empty commit to for keeping notes
	err = new(exec.PipedExec).
		Command(git, "commit", "--allow-empty", "-m", MsgCommitForNotes).
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return err
	}
	// Link notes to it
	if err := AddNotes(wd, notes); err != nil {
		return err
	}

	// Push notes to origin with retry
	err = utils.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, push, origin, utils.RefsNotes).
			WorkingDir(wd).
			RunToStrings()

		return err
	})
	if err != nil {
		logger.Verbose(stderr)

		return fmt.Errorf("failed to push notes to origin: %w", err)
	}
	utils.DelayIfTest()

	// Push branch to origin with retry
	err = utils.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, push, "-u", origin, branchName).
			WorkingDir(wd).
			RunToStrings()

		return err
	})
	if err != nil {
		logger.Verbose(stderr)

		return fmt.Errorf("failed to push branch to origin: %w, stdout: %s", err, stdout)
	}

	utils.DelayIfTest()

	return nil
}

// normalizeBranchName normalizes a branch name to comply with Git branch naming rules.
// It replaces all invalid characters with dashes and ensures the name follows Git conventions.
// Rules applied:
// - Replace invalid characters (ASCII control chars, ~, ^, :, ?, [, *, spaces, ..) with dash
// - Remove leading/trailing dots, slashes, and dashes
// - Replace consecutive dashes with a single dash
// - Ensure lowercase for consistency
func normalizeBranchName(branchName string) string {
	if branchName == "" {
		return branchName
	}

	// Define characters that should be replaced with dash
	invalidChars := []string{
		" ", "~", "^", ":", "?", "[", "]", "*", "\\", "#", "!",
		"\t", "\n", "\r", // whitespace characters
		"\x00", "\x01", "\x02", "\x03", "\x04", "\x05", "\x06", "\x07", // ASCII control chars
		"\x08", "\x09", "\x0a", "\x0b", "\x0c", "\x0d", "\x0e", "\x0f",
		"\x10", "\x11", "\x12", "\x13", "\x14", "\x15", "\x16", "\x17",
		"\x18", "\x19", "\x1a", "\x1b", "\x1c", "\x1d", "\x1e", "\x1f",
		"\x7f", // DEL character
	}

	// Replace invalid characters with dash
	normalized := branchName
	for _, char := range invalidChars {
		normalized = strings.ReplaceAll(normalized, char, "-")
	}

	// Replace ".." (double dot) with single dash
	normalized = strings.ReplaceAll(normalized, "..", "-")

	// Remove leading dots
	normalized = strings.TrimLeft(normalized, ".")

	// Remove trailing and leading slashes
	normalized = strings.Trim(normalized, "/")

	// Remove trailing dash
	normalized = strings.TrimRight(normalized, "-")

	// Remove leading and trailing dashes and underscores
	normalized = strings.Trim(normalized, "-_")

	// Replace consecutive dashes with a single dash
	re := regexp.MustCompile(`-+`)
	normalized = re.ReplaceAllString(normalized, "-")

	return normalized
}
