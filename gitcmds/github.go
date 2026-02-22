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

// CreateGithubLinkToIssue create a link between an upstream GitHub issue and the dev branch
func CreateGithubLinkToIssue(wd, parentRepo, githubIssueURL string, issueNumber int, args ...string) (branch string, notes []string, err error) {
	repo, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return "", nil, fmt.Errorf("GetRepoAndOrgName failed: %w", err)
	}

	if len(repo) == 0 {
		return "", nil, errors.New(repoNotFound)
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
			return "", nil, errors.New(stderr)
		}

		return "", nil, fmt.Errorf("failed to set default repo: %w", err)
	}
	printLn(stdout)

	branchName, err := buildDevBranchName(githubIssueURL)
	if err != nil {
		return "", nil, err
	}

	// check if a branch already exists in remote
	stdout, stderr, err = new(exec.PipedExec).
		Command(git, "ls-remote", "--heads", origin, branch).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", nil, errors.New(stderr)
		}

		return "", nil, fmt.Errorf("failed to check if branch exists in origin remote: %w", err)
	}

	if len(stdout) > 0 {
		return "", nil, fmt.Errorf("branch %s already exists in origin remote", branch)
	}

	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return "", nil, fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	stdout, stderr, err = new(exec.PipedExec).
		Command("gh", "issue", "develop", strIssueNum, "--branch-repo="+myrepo, "--repo="+parentRepo, "--name="+branchName, "--base="+mainBranch).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", nil, errors.New(stderr)
		}

		return "", nil, fmt.Errorf("failed to create development branch for issue: %w", err)
	} // delay to ensure branch is created
	logger.Verbose(stdout)

	utils.DelayIfTest()

	branch = strings.TrimSpace(stdout)
	segments := strings.Split(branch, slash)
	branch = segments[len(segments)-1]

	if len(branch) == 0 {
		return "", nil, errors.New("can not create branch for issue")
	}
	// old-style notes
	issueName, err := GetGithubIssueNameByNumber(strIssueNum, parentRepo)
	if err != nil {
		return "", nil, err
	}

	comment := IssuePRTtilePrefix + " '" + issueName + "' "
	body := ""
	if len(issueName) > 0 {
		body = IssueSign + strIssueNum + oneSpace + issueName
	}
	// Prepare new notes with issue name as description
	notesObj, err := notesPkg.Serialize(githubIssueURL, "", notesPkg.BranchTypeDev, issueName)
	if err != nil {
		return "", nil, err
	}

	return branch, []string{comment, body, notesObj}, nil
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

func buildDevBranchName(issueURL string) (string, error) {
	// Extract issue number from URL
	parts := strings.Split(issueURL, slash)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid issue URL format: %s", issueURL)
	}
	issueNumber := parts[len(parts)-1]

	// Extract owner and repo from URL
	repoURL := strings.Split(issueURL, "/issues/")[0]
	urlParts := strings.Split(repoURL, slash)
	if len(urlParts) < 5 { //nolint:revive
		return "", fmt.Errorf("invalid GitHub URL format: %s", repoURL)
	}
	owner := urlParts[3] //nolint:revive
	repo := urlParts[4]  //nolint:revive

	// Use gh CLI to get issue title
	stdout, stderr, err := new(exec.PipedExec).
		Command("gh", "issue", "view", issueNumber, "--repo", fmt.Sprintf("%s/%s", owner, repo), "--json", "title").
		RunToStrings()

	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", errors.New(stderr)
		}

		return "", fmt.Errorf("failed to get issue title: %w", err)
	}
	logger.Verbose(stdout)

	// Parse JSON response
	var issueData struct {
		Title string `json:"title"`
	}

	if err := json.Unmarshal([]byte(stdout), &issueData); err != nil {
		return "", fmt.Errorf("failed to parse issue data: %w", err)
	}

	// Create kebab-case version of the title
	kebabTitle := strings.ToLower(issueData.Title)
	// Replace spaces and special characters with dashes
	kebabTitle = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(kebabTitle, "-")
	// Remove leading and trailing dashes
	kebabTitle = strings.Trim(kebabTitle, "-")

	// Construct branch name: {issue-number}-{kebab-case-title}
	branchName := fmt.Sprintf("%s-%s", issueNumber, kebabTitle)

	// Ensure branch name doesn't exceed git's limit (usually around 250 chars)
	if len(branchName) > maximumBranchNameLength {
		branchName = branchName[:maximumBranchNameLength]
	}
	branchName = utils.CleanArgFromSpecSymbols(branchName)
	// Add suffix "-dev" for a dev branch
	branchName += "-dev"

	return branchName, nil
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
