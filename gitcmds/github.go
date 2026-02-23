/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package gitcmds

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	notesPkg "github.com/untillpro/qs/internal/notes"
	"github.com/untillpro/qs/utils"
)

// LinkBranchToGithubIssue links an existing remote branch to a GitHub issue and prepares notes.
// The branch must already exist on the remote before calling this function.
func LinkBranchToGithubIssue(wd, parentRepo, githubIssueURL string, issueNumber int, branchName string, args ...string) (notes []string, err error) {
	repo, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return nil, fmt.Errorf("GetRepoAndOrgName failed: %w", err)
	}

	if len(repo) == 0 {
		return nil, errors.New(repoNotFound)
	}

	strIssueNum := strconv.Itoa(issueNumber)
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
		Command("gh", "issue", "develop", strIssueNum, "--branch-repo="+myrepo, "--repo="+parentRepo, "--name="+branchName, "--base="+mainBranch).
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

func BuildDevBranchName(issueURL string) (string, []string, error) {
	parts := strings.Split(issueURL, slash)
	if len(parts) < 2 {
		return "", nil, fmt.Errorf("invalid issue URL format: %s", issueURL)
	}
	issueNumber := parts[len(parts)-1]

	repoURL := strings.Split(issueURL, "/issues/")[0]
	urlParts := strings.Split(repoURL, slash)
	if len(urlParts) < 5 { //nolint:revive
		return "", nil, fmt.Errorf("invalid GitHub URL format: %s", repoURL)
	}
	owner := urlParts[3] //nolint:revive
	repo := urlParts[4]  //nolint:revive

	stdout, stderr, err := new(exec.PipedExec).
		Command("gh", "issue", "view", issueNumber, "--repo", fmt.Sprintf("%s/%s", owner, repo), "--json", "title").
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)
		if len(stderr) > 0 {
			return "", nil, errors.New(stderr)
		}
		return "", nil, fmt.Errorf("failed to get issue title: %w", err)
	}
	logger.Verbose(stdout)

	var issueData struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(stdout), &issueData); err != nil {
		return "", nil, fmt.Errorf("failed to parse issue data: %w", err)
	}

	kebabTitle := strings.ToLower(issueData.Title)
	kebabTitle = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(kebabTitle, "-")
	kebabTitle = strings.Trim(kebabTitle, "-")

	branchName := fmt.Sprintf("%s-%s", issueNumber, kebabTitle)
	if len(branchName) > maximumBranchNameLength {
		branchName = branchName[:maximumBranchNameLength]
	}
	branchName = utils.CleanArgFromSpecSymbols(branchName)
	branchName += "-dev"

	comment := IssuePRTtilePrefix + " '" + issueData.Title + "' "
	body := ""
	if len(issueData.Title) > 0 {
		body = IssueSign + issueNumber + oneSpace + issueData.Title
	}
	notesObj, err := notesPkg.Serialize(issueURL, "", notesPkg.BranchTypeDev, issueData.Title)
	if err != nil {
		return "", nil, err
	}

	return branchName, []string{comment, body, notesObj}, nil
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
