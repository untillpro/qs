/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package utils

import (
	"fmt"
	"os"

	osExec "os/exec"

	"github.com/untillpro/goutils/exec"
)

func CheckGH() error {
	if !ghInstalled() {
		return fmt.Errorf("Github cli utility 'gh' is not installed.\nTo install visit page https://cli.github.com/")
	}
	if !ghLoggedIn() {
		return fmt.Errorf("GH utility is not logged in")
	}
	return nil
}

// ghInstalled returns is gh utility installed
func ghInstalled() bool {
	_, _, err := new(exec.PipedExec).
		Command("gh", "--version").
		RunToStrings()
	return err == nil
}

// ghLoggedIn returns is gh logged in
func ghLoggedIn() bool {
	_, _, err := new(exec.PipedExec).
		Command("gh", "auth", "status").
		RunToStrings()
	return err == nil
}

// VerifyGitHubRepoExists checks if a GitHub repository exists and is accessible
func VerifyGitHubRepoExists(owner, repo, token string) error {
	//nolint:gosec
	cmd := osExec.Command("gh", "repo", "view", fmt.Sprintf("%s/%s", owner, repo))

	// Only set GITHUB_TOKEN if a token is provided, otherwise use current gh auth
	if token != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", token))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("repository %s/%s not accessible: %w, output: %s", owner, repo, err, output)
	}

	return nil
}
