/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package gitcmds

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/utils"
)

// LinkBranchToGithubIssue links an existing remote branch to a GitHub issue and prepares notes.
// The branch must already exist on the remote before calling this function.
func LinkBranchToGithubIssue(wd, parentRepo, githubIssueURL, issueNumber, branchName string, args ...string) (notes []string, err error) {
	repo, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return nil, fmt.Errorf("GetRepoAndOrgName failed: %w", err)
	}

	if len(repo) == 0 {
		return nil, errors.New(repoNotFound)
	}

	myrepo := org + slash + repo

	if len(args) > 0 {
		issueURL := args[0]
		issueRepo := GetGithubIssueRepoFromURL(issueURL)
		if len(issueRepo) > 0 {
			parentRepo = issueRepo
		}
	}

	stdout, stderr, err := new(exec.PipedExec).
		Command("gh", "repo", "set-default", myrepo).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return nil, errors.New(stderr)
		}

		return nil, fmt.Errorf("failed to set default repo: %w", err)
	}
	printLn(stdout)

	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return nil, fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	stdout, stderr, err = new(exec.PipedExec).
		Command("gh", "issue", "develop", issueNumber, "--branch-repo="+myrepo, "--repo="+parentRepo, "--name="+branchName, "--base="+mainBranch).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return nil, errors.New(stderr)
		}

		return nil, fmt.Errorf("failed to link branch to issue: %w", err)
	}
	logger.Verbose(stdout)

	utils.DelayIfTest()

	return nil, nil
}

func GetGithubIssueRepoFromURL(url string) (repoName string) {
	if len(url) < 2 {
		return
	}
	if strings.HasSuffix(url, slash) {
		url = url[:len(url)-1]
	}

	arr := strings.Split(url, slash)
	if len(arr) > issuelineLength {
		repo := arr[len(arr)-issuelinePosRepo]
		org := arr[len(arr)-issuelinePosOrg]
		repoName = org + slash + repo
	}

	return
}

func GetGithubIssueNameByNumber(issueNum string, parentrepo string) (string, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command("gh", "issue", "view", issueNum, "--repo", parentrepo, "--json", "title").
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", errors.New(stderr)
		}

		return "", fmt.Errorf("failed to get issue name by number: %w", err)
	}

	type issueDetails struct {
		Title string `json:"title"`
	}

	var issue issueDetails
	if err := json.Unmarshal([]byte(stdout), &issue); err != nil {
		return "", fmt.Errorf("failed to parse issue title JSON: %w", err)
	}

	return issue.Title, nil
}
