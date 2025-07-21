package gitcmds

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	osExec "os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	goGitPkg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	contextPkg "github.com/untillpro/qs/internal/context"
	"github.com/untillpro/qs/internal/helper"
	notesPkg "github.com/untillpro/qs/internal/notes"
	"github.com/untillpro/qs/utils"
)

const (
	mimm              = "-m"
	slash             = "/"
	caret             = "\n"
	git               = "git"
	push              = "push"
	pull              = "pull"
	fetch             = "fetch"
	branch            = "branch"
	origin            = "origin"
	originSlash       = "origin/"
	httppref          = "https"
	pushYes           = "y"
	nochecksmsg       = "no checks reported"
	msgWaitingPR      = "Waiting PR checks.."
	MsgPreCommitError = "Attempt to commit too"
	MsgCommitForNotes = "Commit for keeping notes in branch"
	oneSpace          = " "
	err128            = "128"

	repoNotFound            = "git repo name not found"
	userNotFound            = "git user name not found"
	ErrAlreadyForkedMsg     = "you are in fork already\nExecute 'qs dev [branch name]' to create dev branch"
	ErrMsgPRNotesImpossible = "pull request without comments is impossible"
	ErrTimer40Sec           = "time out 40 seconds"
	ErrSomethigWrong        = "something went wrong"
	PushDefaultMsg          = "dev"

	IssuePRTtilePrefix = "Resolves issue"
	IssueSign          = "Resolves #"

	prTimeWait                     = 40
	minIssueNoteLength             = 10
	minRepoNameLength              = 4
	bashFilePerm       os.FileMode = 0644

	issuelineLength  = 5
	issuelinePosOrg  = 4
	issuelinePosRepo = 3
)

type gchResponse struct {
	_stdout string
	_stderr string
	_err    error
}

func CheckIfGitRepo(wd string) (string, error) {
	stdout, _, err := new(exec.PipedExec).
		Command("git", "status", "-s").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		if strings.Contains(err.Error(), err128) {
			err = errors.New("this is not a git repository")
		}
	}

	return stdout, err
}

// ChangedFilesExist s.e.
func ChangedFilesExist(wd string) (string, bool, error) {
	files, err := CheckIfGitRepo(wd)
	uncommitedFiles := strings.TrimSpace(files)

	return uncommitedFiles, len(uncommitedFiles) > 0, err
}

// stashEntriesExist checks if there are any stash entries in the git repository
func stashEntriesExist(wd string) (bool, error) {
	stdout, _, err := new(exec.PipedExec).
		Command(git, "stash", "list").
		WorkingDir(wd).
		RunToStrings()
	stashEntries := strings.TrimSpace(stdout)

	return len(stashEntries) > 0, err
}

// Status shows git repo status
func Status(wd string) error {
	stdout, stderr, err := new(exec.PipedExec).
		Command("git", "remote", "-v").
		WorkingDir(wd).
		Command("grep", fetch).
		Command("sed", "s/(fetch)//").
		RunToStrings()
	if err != nil {
		if len(stderr) > 0 {
			logger.Verbose(stderr)
		}
		if strings.Contains(err.Error(), err128) {
			return errors.New("this is not a git repository")
		}
	}

	if err != nil {
		return fmt.Errorf("git remote -v failed: %w", err)
	}
	logger.Verbose(stdout)

	return new(exec.PipedExec).
		Command("git", "status", "-s", "-b", "-uall").
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
}

/*
	- Pull
	- Get current verson
	- If PreRelease is not empty fails
	- Calculate target version
	- Ask
	- Save version
	- Commit
	- Tag with target version
	- Bump current version
	- Commit
	- Push commits and tags
*/

// Release current branch. Remove PreRelease, tag, bump version, push
func Release(wd string) error {

	// *************************************************
	fmt.Fprintln(os.Stdout, "Pulling")
	err := new(exec.PipedExec).
		Command("git", pull).
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return err
	}

	// *************************************************
	fmt.Fprintln(os.Stdout, "Reading current version")
	currentVersion, err := utils.ReadVersion()
	if err != nil {
		return fmt.Errorf("Error reading file 'version': %w", err)
	}
	if len(currentVersion.PreRelease) <= 0 {
		return errors.New("pre-release part of version does not exist: " + currentVersion.String())
	}

	// Calculate target version

	targetVersion := currentVersion
	targetVersion.PreRelease = ""

	fmt.Printf("Version %v will be tagged, bumped and pushed, agree? [y]", targetVersion)
	var response string
	_, _ = fmt.Scanln(&response)
	if response != "y" {
		return errors.New("release aborted by user")
	}

	// *************************************************
	fmt.Fprintln(os.Stdout, "Updating 'version' file")
	if err := targetVersion.Save(); err != nil {
		return fmt.Errorf("Error saving file 'version': %w", err)
	}

	// *************************************************
	fmt.Fprintln(os.Stdout, "Committing target version")
	{
		params := []string{"commit", "-a", mimm, "#scm-ver " + targetVersion.String()}
		err = new(exec.PipedExec).
			Command(git, params...).
			WorkingDir(wd).
			Run(os.Stdout, os.Stdout)
		if err != nil {
			return err
		}
	}

	// *************************************************
	fmt.Fprintln(os.Stdout, "Tagging")
	{
		tagName := "v" + targetVersion.String()
		n := time.Now()
		params := []string{"tag", mimm, "Version " + tagName + " of " + n.Format("2006/01/02 15:04:05"), tagName}
		err = new(exec.PipedExec).
			Command(git, params...).
			WorkingDir(wd).
			Run(os.Stdout, os.Stdout)
		if err != nil {
			return err
		}
	}

	// *************************************************
	fmt.Fprintln(os.Stdout, "Bumping version")
	newVersion := currentVersion
	{
		newVersion.Minor++
		newVersion.PreRelease = "SNAPSHOT"
		if err := targetVersion.Save(); err != nil {
			return fmt.Errorf("Error saving file 'version': %w", err)
		}
	}

	// *************************************************
	fmt.Fprintln(os.Stdout, "Committing new version")
	{
		params := []string{"commit", "-a", mimm, "#scm-ver " + newVersion.String()}
		err = new(exec.PipedExec).
			Command(git, params...).
			WorkingDir(wd).
			Run(os.Stdout, os.Stdout)
		if err != nil {
			return err
		}
	}

	// *************************************************
	fmt.Fprintln(os.Stdout, "Pushing to origin")
	{
		params := []string{push, "--follow-tags", origin}
		err = helper.Retry(func() error {
			return new(exec.PipedExec).
				Command(git, params...).
				WorkingDir(wd).
				Run(os.Stdout, os.Stdout)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// Upload uploads sources to git repo
func Upload(cmd *cobra.Command, wd string) error {
	commitMessageParts := cmd.Context().Value(contextPkg.CtxKeyCommitMessage).([]string)

	err := new(exec.PipedExec).
		Command(git, "add", ".").
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return err
	}

	params := []string{"commit", "-a"}
	for _, m := range commitMessageParts {
		params = append(params, mimm, m)
	}
	_, stderr, err := new(exec.PipedExec).
		Command(git, params...).
		WorkingDir(wd).
		RunToStrings()
	if strings.Contains(stderr, MsgPreCommitError) {
		var response string
		fmt.Println("")
		fmt.Println(strings.TrimSpace(stderr))
		fmt.Print("Do you want to commit anyway(y/n)?")
		_, _ = fmt.Scanln(&response)
		if response != "y" {
			return nil
		}
		params = append(params, "-n")
		err = new(exec.PipedExec).
			Command(git, params...).
			WorkingDir(wd).
			Run(os.Stdout, os.Stdout)
	}
	if err != nil {
		return err
	}

	// make pull before push
	_, _, err = new(exec.PipedExec).
		Command(git, pull).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return fmt.Errorf("error pulling before push: %w", err)
	}

	brName, err := GetCurrentBranchName(wd)
	if err != nil {
		return err
	}

	// Push notes to origin
	err = helper.Retry(func() error {
		return new(exec.PipedExec).
			Command(git, push, origin, "refs/notes/*:refs/notes/*").
			WorkingDir(wd).
			Run(os.Stdout, os.Stdout)
	})
	if err != nil {
		return err
	}

	if helper.IsTest() {
		helper.Delay()
	}

	// Push branch to origin
	var stdout string

	// Check if branch already has upstream tracking
	hasUpstream, err := hasUpstreamBranch(wd, brName)
	if err != nil {
		return fmt.Errorf("failed to check upstream branch: %w", err)
	}

	// Only use -u flag if upstream is not already configured
	pushArgs := []string{push, origin, brName}
	if !hasUpstream {
		pushArgs = []string{push, "-u", origin, brName}
	}

	err = helper.Retry(func() error {
		var pushErr error
		stdout, stderr, pushErr = new(exec.PipedExec).
			Command(git, pushArgs...).
			WorkingDir(wd).
			RunToStrings()
		return pushErr
	})
	if err != nil {
		logger.Verbose(stderr)
		return err
	}
	logger.Verbose(stdout)

	return nil
}

// Stash stashes uncommitted changes
func Stash(wd string) error {
	_, stderr, err := new(exec.PipedExec).
		Command("git", "stash").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return fmt.Errorf("git stash failed: %v - %s", err, stderr)
	}

	return nil
}

// Unstash pops the latest stash
func Unstash(wd string) error {
	stdout, stderr, err := new(exec.PipedExec).
		Command("git", "stash", "pop").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		const msg = "No stash entries found"
		if strings.Contains(stdout, msg) || strings.Contains(stderr, msg) {
			return nil // No stash to pop, return nil
		}

		return fmt.Errorf("git stash pop failed: %v", err)
	}

	return nil
}

func HaveUncommittedChanges(wd string) (bool, error) {
	output, _, err := new(exec.PipedExec).
		Command(git, "status", "--porcelain").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return false, fmt.Errorf("failed to check if there are uncommitted changes: %w", err)
	}

	return len(output) > 0, nil
}

// Download sources from git repo
func Download(wd string) error {
	uncommittedChanges, err := HaveUncommittedChanges(wd)
	if err != nil {
		return err
	}

	if uncommittedChanges {
		return errors.New("There are uncommitted changes in the repository.")
	}

	branchName, isMain, err := IamInMainBranch(wd)
	if err != nil {
		return err
	}
	// pull from origin for dev branch
	if !isMain {
		// Fetch notes from origin
		err := helper.Retry(func() error {
			return new(exec.PipedExec).
				Command(git, fetch, "origin", "--force", "refs/notes/*:refs/notes/*").
				WorkingDir(wd).
				Run(os.Stdout, os.Stdout)
		})
		if err != nil {
			return err
		}

		return helper.Retry(func() error {
			return new(exec.PipedExec).
				Command(git, pull, "origin").
				WorkingDir(wd).
				Run(os.Stdout, os.Stdout)
		})
	}

	// pull from upstream if exists and current branch is main
	if !UpstreamNotExist(wd) {
		err := helper.Retry(func() error {
			return new(exec.PipedExec).
				Command(git, pull, "upstream", branchName).
				WorkingDir(wd).
				Run(os.Stdout, os.Stdout)
		})

		return err
	}

	return nil
}

// Gui shows gui
func Gui(wd string) error {
	return new(exec.PipedExec).
		Command(git, "gui").
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
}

func getFullRepoAndOrgName(wd string) (string, error) {
	stdout, _, err := new(exec.PipedExec).
		Command(git, "config", "--local", "remote.origin.url").
		WorkingDir(wd).
		Command("sed", "s/\\.git$//").
		RunToStrings()
	if err != nil {
		return "", fmt.Errorf("getFullRepoAndOrgName failed: %w", err)
	}

	return strings.TrimSuffix(strings.TrimSpace(stdout), slash), err
}

// GetRepoAndOrgName - from .git/config
func GetRepoAndOrgName(wd string) (repo string, org string, err error) {
	repoURL, err := getFullRepoAndOrgName(wd)
	if err != nil {
		return "", "", err
	}

	org, repo, _, err = ParseGitRemoteURL(repoURL)
	if err != nil {
		return "", "", err
	}

	return
}

// ParseGitRemoteURL extracts account, repository name, and token from a git remote URL.
// Handles HTTPS format (https://github.com/account/repo.git),
// and HTTPS with token (https://account:token@github.com/account/repo.git or https://oauth2:token@github.com/repo.git)
func ParseGitRemoteURL(remoteURL string) (account, repo string, token string, err error) {
	if remoteURL == "" {
		return "", "", "", errors.New("remote URL is empty")
	}

	remoteURL = strings.TrimSuffix(remoteURL, ".git")
	// Handle HTTPS format: https://github.com/account/repo
	// or https://account:token@github.com/account/repo
	if strings.HasPrefix(remoteURL, "http") {
		u, err := url.Parse(remoteURL)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to parse URL: %w", err)
		}

		// Extract token if present in the userinfo section
		if u.User != nil {
			password, hasPassword := u.User.Password()
			if hasPassword {
				token = password
			}
		}

		// Remove leading '/' if any
		path := strings.TrimPrefix(u.Path, "/")
		pathParts := strings.Split(path, "/")

		if len(pathParts) < 2 {
			return "", "", "", fmt.Errorf("invalid repository path in URL: %s", remoteURL)
		}

		// If no token was found or username is oauth2 (common for token auth without real account name)
		// use the first path component as account
		if u.User != nil && u.User.Username() == "oauth2" {
			account = pathParts[0]
		} else if u.User != nil && u.User.Username() != "" {
			account = u.User.Username()
		} else {
			account = pathParts[0]
		}

		return account, pathParts[1], token, nil
	}

	return "", "", "", fmt.Errorf("unsupported git URL format: %s", remoteURL)
}

func IsMainOrg(wd string) (bool, error) {
	_, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return false, err
	}
	userName, err := getUserName(wd)

	return org != userName, err
}

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
		if err := new(exec.PipedExec).
			Command(git, "add", ".").
			WorkingDir(wd).
			Run(os.Stdout, os.Stdout); err != nil {
			return repo, err
		}

		if err := new(exec.PipedExec).
			Command(git, "stash").
			WorkingDir(wd).
			Run(os.Stdout, os.Stdout); err != nil {
			return repo, err
		}
	}

	err = helper.Retry(func() error {
		return new(exec.PipedExec).
			Command("gh", "repo", "fork", org+slash+repo, "--clone=false").
			WorkingDir(wd).
			Run(os.Stdout, os.Stdout)
	})
	if err != nil {
		logger.Verbose("Fork error:", err)
		return repo, err
	}

	// Get current user name to verify fork
	userName, err := getUserName(wd)
	if err != nil {
		logger.Verbose("Failed to get user name for verification:", err)
		return repo, err
	}

	// Verify fork was created and is accessible with retry
	err = helper.Retry(func() error {
		// Try to get user email to get a valid token context, then verify repo
		userEmail, emailErr := GetUserEmail()
		if emailErr != nil {
			return fmt.Errorf("failed to verify GitHub authentication: %w", emailErr)
		}
		logger.Verbose("Verified GitHub authentication for user: %s", userEmail)

		// Verify the forked repository exists and is accessible
		return helper.VerifyGitHubRepoExists(userName, repo, "")
	})
	if err != nil {
		logger.Verbose("Fork verification failed:", err)
		return repo, fmt.Errorf("fork verification failed: %w", err)
	}

	fmt.Fprintln(os.Stdout, "Fork created and verified successfully")
	return repo, nil
}

// GetUserEmail - github user email
func GetUserEmail() (string, error) {
	var stdout string
	err := helper.Retry(func() error {
		var apiErr error
		stdout, _, apiErr = new(exec.PipedExec).
			Command("gh", "api", "user", "--jq", ".email").
			RunToStrings()
		return apiErr
	})

	return strings.TrimSpace(stdout), err
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

func PopStashedFiles(wd string) error {
	if ok, err := stashEntriesExist(wd); !ok {
		return err
	}

	_, stderr, err := new(exec.PipedExec).
		Command(git, "stash", "pop").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return fmt.Errorf("PopStashedFiles error: %s", stderr)
	}

	return nil
}

func GetMainBranch(wd string) (string, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, branch, "-r").
		WorkingDir(wd).
		Command("grep", "-E", "(/main|/master)([^a-zA-Z0-9]|$)").
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)
		logger.Verbose(stdout)

		return "", err
	}

	// Check if the output contains "main" or "master"
	mainBranchFound := strings.Contains(stdout, "/main")
	masterBranchFound := strings.Contains(stdout, "/master")

	switch {
	case mainBranchFound && masterBranchFound:
		return "", fmt.Errorf("both main and master branches found")
	case mainBranchFound:
		return "main", nil
	case masterBranchFound:
		return "master", nil
	}

	return "", errors.New("neither main nor master branches found")
}

func getUserName(wd string) (string, error) {
	stdout, _, err := new(exec.PipedExec).
		Command("gh", "api", "user").
		WorkingDir(wd).
		Command("jq", "-r", ".login").
		RunToStrings()

	return strings.TrimSpace(stdout), err
}

func MakeUpstreamForBranch(wd string, parentRepo string) error {
	_, _, err := new(exec.PipedExec).
		Command(git, "remote", "add", "upstream", "https://github.com/"+parentRepo).
		WorkingDir(wd).
		RunToStrings()

	return err
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
		return fmt.Errorf("failed to get main branch: %w", err)
	}

	err = new(exec.PipedExec).
		Command(git, "remote", "rename", "origin", "upstream").
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return err
	}

	err = new(exec.PipedExec).
		Command(git, "remote", "add", "origin", "https://github.com/"+userName+slash+repo).
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return err
	}
	// delay to ensure remote is added
	if helper.IsTest() {
		helper.Delay()
	}

	err = helper.Retry(func() error {
		return new(exec.PipedExec).
			Command(git, "fetch", "origin").
			WorkingDir(wd).
			Run(os.Stdout, os.Stdout)
	})
	if err != nil {
		return err
	}

	return new(exec.PipedExec).
		Command(git, branch, "--set-upstream-to", originSlash+mainBranch, mainBranch).
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
}

func GetIssueRepoFromURL(url string) (repoName string) {
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

// DevIssue create link between upstream Guthub issue and dev branch
func DevIssue(wd, parentRepo, githubIssueURL string, issueNumber int, args ...string) (branch string, notes []string, err error) {
	repo, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return "", nil, fmt.Errorf("GetRepoAndOrgName failed: %w", err)
	}

	if len(repo) == 0 {
		return "", nil, errors.New(repoNotFound)
	}

	strIssueNum := strconv.Itoa(issueNumber)
	myrepo := org + slash + repo
	if err != nil {
		return "", nil, err
	}

	if len(args) > 0 {
		issueURL := args[0]
		issueRepo := GetIssueRepoFromURL(issueURL)
		if len(issueRepo) > 0 {
			parentRepo = issueRepo
		}
	}

	err = new(exec.PipedExec).
		Command("gh", "repo", "set-default", myrepo).
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return "", nil, err
	}

	branchName, err := buildDevBranchName(githubIssueURL)
	if err != nil {
		return "", nil, err
	}

	// check if branch already exists in remote
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "ls-remote", "--heads", "origin", branch).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		return "", nil, err
	}
	if len(stdout) > 0 {
		return "", nil, fmt.Errorf("branch %s already exists in origin remote", branch)
	}

	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get main branch: %w", err)
	}

	stdout, stderr, err = new(exec.PipedExec).
		Command("gh", "issue", "develop", strIssueNum, "--branch-repo="+myrepo, "--repo="+parentRepo, "--name="+branchName, "--base="+mainBranch).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)
		return "", nil, err
	}
	// delay to ensure branch is created
	if helper.IsTest() {
		helper.Delay()
	}

	branch = strings.TrimSpace(stdout)
	segments := strings.Split(branch, slash)
	branch = segments[len(segments)-1]

	if len(branch) == 0 {
		return "", nil, errors.New("Can not create branch for issue")
	}
	// old-style notes
	issueName := GetIssueNameByNumber(strIssueNum, parentRepo)
	comment := IssuePRTtilePrefix + " '" + issueName + "' "
	body := ""
	if len(issueName) > 0 {
		body = IssueSign + strIssueNum + oneSpace + issueName
	}
	// Prepare new notes
	notesObj, err := notesPkg.Serialize(githubIssueURL, "", notesPkg.BranchTypeDev)
	if err != nil {
		return "", nil, err
	}

	return branch, []string{comment, body, notesObj}, nil
}

// SyncMainBranch syncs the local main branch with upstream and origin
// Flow:
// 1. Pull from UpstreamMain to MainBranch with rebase
// 2. If upstream exists
// - Pull from origin to MainBranch with rebase
// - Push to origin from MainBranch
func SyncMainBranch(wd string) error {
	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return fmt.Errorf("failed to get main branch: %w", err)
	}

	// Pull from UpstreamMain to MainBranch with rebase
	remoteUpstreamURL := GetRemoteUpstreamURL(wd)

	if len(remoteUpstreamURL) > 0 {
		stdout, stderr, err := new(exec.PipedExec).
			Command(git, pull, "--rebase", "upstream", mainBranch, "--no-edit").
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			return fmt.Errorf("failed to pull from upstream/main: %w, stdout: %s", err, stdout)
		}
		logger.Verbose(stdout)
	}

	// Pull from origin to MainBranch with rebase
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, pull, "--rebase", "origin", mainBranch, "--no-edit").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		return fmt.Errorf("failed to pull from origin/main: %w, stdout: %s", err, stdout)
	}
	logger.Verbose(stdout)

	// Push to origin from MainBranch
	err = helper.Retry(func() error {
		var pushErr error
		stdout, stderr, pushErr = new(exec.PipedExec).
			Command(git, push, "origin", mainBranch).
			WorkingDir(wd).
			RunToStrings()
		return pushErr
	})
	if err != nil {
		logger.Verbose(stderr)
		return fmt.Errorf("failed to push to origin/main: %w, stdout: %s", err, stdout)
	}
	logger.Verbose(stdout)

	return nil
}

// getBranchTypeByName returns branch type based on branch name
func getBranchTypeByName(branchName string) notesPkg.BranchType {
	switch {
	case strings.HasSuffix(branchName, "-dev"):
		return notesPkg.BranchTypeDev
	case strings.HasSuffix(branchName, "-pr"):
		return notesPkg.BranchTypePr
	default:
		return notesPkg.BranchTypeUnknown
	}
}

func buildDevBranchName(issueURL string) (string, error) {
	// Extract issue number from URL
	parts := strings.Split(issueURL, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid issue URL format: %s", issueURL)
	}
	issueNumber := parts[len(parts)-1]

	// Extract owner and repo from URL
	repoURL := strings.Split(issueURL, "/issues/")[0]
	urlParts := strings.Split(repoURL, "/")
	if len(urlParts) < 5 {
		return "", fmt.Errorf("invalid GitHub URL format: %s", repoURL)
	}
	owner := urlParts[3]
	repo := urlParts[4]

	// Use gh CLI to get issue title
	stdout, stderr, err := new(exec.PipedExec).
		Command("gh", "issue", "view", issueNumber, "--repo", fmt.Sprintf("%s/%s", owner, repo), "--json", "title").
		RunToStrings()

	if err != nil {
		return "", fmt.Errorf("failed to get issue: %w, stderr: %s", err, stderr)
	}

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
	if len(branchName) > 100 {
		branchName = branchName[:100]
	}
	// Add suffix "-dev" for a dev branch
	branchName += "-dev"

	return branchName, nil
}

// GetBranchType returns branch type based on notes or branch name
func GetBranchType(wd string) (notesPkg.BranchType, error) {
	currentBranchName, err := GetCurrentBranchName(wd)
	if err != nil {
		return notesPkg.BranchTypeUnknown, err
	}

	notes, _, err := GetNotes(wd, currentBranchName)
	if err != nil {
		logger.Verbose(err)
	}

	if len(notes) > 0 {
		notesObj, ok := notesPkg.Deserialize(notes)
		if !ok {
			if isOldStyledBranch(notes) {
				return notesPkg.BranchTypeDev, nil
			}
		}

		if notesObj != nil {
			return notesObj.BranchType, nil
		}
	}

	return getBranchTypeByName(currentBranchName), nil
}

// isOldStyledBranch checks if branch is old styled
func isOldStyledBranch(notes []string) bool {
	for _, s := range notes {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			if strings.Contains(s, IssuePRTtilePrefix) || strings.Contains(s, IssueSign) {
				return true
			}
		}
	}

	return false
}

func GetIssueNameByNumber(issueNum string, parentrepo string) string {
	stdouts, _, err := new(exec.PipedExec).
		Command("gh", "issue", "view", issueNum, "--repo", parentrepo).
		Command("grep", "title:").
		Command("gawk", "{ $1=\"\"; print substr($0, 2) }").
		RunToStrings()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdouts)
}

// Dev creates dev branch and pushes it to origin
// Parameters:
// branch - branch name
// notes - notes for branch
// checkRemoteBranchExistence - if true, checks if branch already exists in remote
func Dev(wd, branchName string, notes []string, checkRemoteBranchExistence bool) error {
	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return fmt.Errorf("failed to get main branch: %w", err)
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
			if err != nil {
				return err
			}
		}
	}
	if err != nil {
		return err
	}

	if checkRemoteBranchExistence {
		// check if branch already exists in remote
		stdout, stderr, err := new(exec.PipedExec).
			Command(git, "ls-remote", "--heads", "origin", branchName).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			return err
		}
		logger.Verbose(stdout)

		if len(stdout) > 0 {
			return fmt.Errorf("branch %s already exists in origin remote", branchName)
		}
	}

	err = new(exec.PipedExec).
		Command(git, "checkout", "-B", branchName).
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return err
	}

	// Fetch notes from origin before pushing
	stdout, stderr, err = new(exec.PipedExec).
		Command(git, fetch, "--force", origin, "refs/notes/*:refs/notes/*").
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
	err = helper.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, push, origin, "refs/notes/*:refs/notes/*").
			WorkingDir(wd).
			RunToStrings()

		return err
	})
	if err != nil {
		logger.Verbose(stderr)

		return fmt.Errorf("failed to push notes to origin: %w", err)
	}
	if helper.IsTest() {
		helper.Delay()
	}

	// Push branch to origin with retry
	err = helper.Retry(func() error {
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

	if helper.IsTest() {
		helper.Delay()
	}

	return nil
}

func AddNotes(wd string, notes []string) error {
	if len(notes) == 0 {
		return nil
	}
	// Add new Notes
	for _, s := range notes {
		str := strings.TrimSpace(s)
		if len(str) > 0 {
			err := new(exec.PipedExec).
				Command(git, "notes", "append", "-m", str).
				WorkingDir(wd).
				Run(os.Stdout, os.Stdout)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// GetNotes returns notes for a branch
// Returns:
// - notes
// - revision count
// - error if any
func GetNotes(wd, branchName string) (notes []string, revCount int, err error) {
	mainBranchName, err := GetMainBranch(wd)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get main branch: %w", err)
	}

	// get all revision of the branch which does not belong to main branch
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "rev-list", mainBranchName+".."+branchName).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		return notes, 0, fmt.Errorf("failed to get commit list: %v", err)
	}
	if len(stdout) == 0 {
		return notes, 0, errors.New("error: No commits found in current branch")
	}

	// get all notes from revisions got from a previous step
	revList := strings.Split(strings.TrimSpace(stdout), caret)
	for _, rev := range revList {
		// get notes from each revision
		stdout, stderr, err := new(exec.PipedExec).
			Command(git, "notes", "show", rev).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			if strings.Contains(stderr, "no note found") {
				continue
			}
			logger.Verbose(stderr)

			return notes, len(revList), fmt.Errorf("failed to get notes: %v", err)
		}
		// split notes into lines
		rawNotes := strings.Split(stdout, caret)
		for _, rawNote := range rawNotes {
			note := strings.TrimSpace(rawNote)
			if len(note) > 0 {
				notes = append(notes, note)
			}
		}
	}

	if len(notes) == 0 {
		return notes, len(revList), errors.New("error: No notes found in current branch")
	}

	return notes, len(revList), nil
}

// GetParentRepoName - parent repo of forked
func GetParentRepoName(wd string) (name string, err error) {
	repo, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return "", err
	}

	stdout, _, err := new(exec.PipedExec).
		Command("gh", "api", "repos/"+org+slash+repo, "--jq", ".parent.full_name").
		WorkingDir(wd).
		RunToStrings()

	return strings.TrimSpace(stdout), err
}

// IsBranchInMain Is my branch in main org?
func IsBranchInMain(wd string) (bool, error) {
	repo, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return false, err
	}
	parent, err := GetParentRepoName(wd)

	return (parent == org+slash+repo) || (strings.TrimSpace(parent) == ""), err
}

// GetMergedBranchList returns merged user's branch list
func GetMergedBranchList(wd string) (brlist []string, err error) {
	mbrlist := []string{}
	_, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return nil, fmt.Errorf("GetRepoAndOrgName failed: %w", err)
	}

	repo, err := GetParentRepoName(wd)
	if err != nil {
		return nil, fmt.Errorf("GetParentRepoName failed: %w", err)
	}

	stdout, _, err := new(exec.PipedExec).
		Command("gh", "pr", "list", "-L", "200", "--state", "merged", "--author", org, "--repo", repo).
		WorkingDir(wd).
		Command("gawk", "-F:", "{print $2}").
		Command("gawk", "/MERGED/{print $1}").
		RunToStrings()
	if err != nil {
		return []string{}, err
	}

	mbrlistraw := strings.Split(stdout, caret)
	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return nil, fmt.Errorf("failed to get main branch: %w", err)
	}

	curbr, err := GetCurrentBranchName(wd)
	if err != nil {
		return nil, err
	}

	for _, mbranchstr := range mbrlistraw {
		arrstr := strings.TrimSpace(mbranchstr)
		if (strings.TrimSpace(arrstr) != "") && !strings.Contains(arrstr, curbr) && !strings.Contains(arrstr, mainBranch) {
			mbrlist = append(mbrlist, arrstr)
		}
	}
	_, _, err = new(exec.PipedExec).
		Command(git, "remote", "prune", origin).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return nil, err
	}

	stdout, _, err = new(exec.PipedExec).
		Command(git, branch, "-r").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return nil, err
	}
	mybrlist := strings.Split(stdout, caret)

	for _, mybranch := range mybrlist {
		mybranch := strings.TrimSpace(mybranch)
		mybranch = strings.ReplaceAll(strings.TrimSpace(mybranch), originSlash, "")
		mybranch = strings.TrimSpace(mybranch)
		bfound := false
		if !strings.Contains(mybranch, mainBranch) && !strings.Contains(mybranch, "HEAD") {
			for _, mbranch := range mbrlist {
				mbranch = strings.ReplaceAll(strings.TrimSpace(mbranch), "MERGED", "")
				mbranch = strings.TrimSpace(mbranch)
				if mybranch == mbranch {
					bfound = true
					break
				}
			}
		}
		if bfound {
			// delete branch in fork
			brlist = append(brlist, mybranch)
		}
	}

	return brlist, nil
}

// DeleteBranchesRemote delete branch list
func DeleteBranchesRemote(wd string, brs []string) error {
	if len(brs) == 0 {
		return nil
	}

	for _, br := range brs {
		err := helper.Retry(func() error {
			_, _, deleteErr := new(exec.PipedExec).
				Command(git, push, origin, ":"+br).
				WorkingDir(wd).
				RunToStrings()
			return deleteErr
		})
		if err != nil {
			return fmt.Errorf("Branch %s was not deleted: %w", br, err)
		}

		fmt.Printf("Branch %s deleted\n", br)
	}

	return nil
}

func PullUpstream(wd string) error {
	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return fmt.Errorf("failed to get main branch: %w", err)
	}

	err = new(exec.PipedExec).
		Command(git, pull, "-q", "upstream", mainBranch, "--no-edit").
		Run(os.Stdout, os.Stdout)

	if err != nil {
		parentRepoName, err := GetParentRepoName(wd)
		if err != nil {
			return fmt.Errorf("GetParentRepoName failed: %w", err)
		}

		return MakeUpstreamForBranch(wd, parentRepoName)
	}

	return nil
}

// GetGoneBranchesLocal returns gone local branches
func GetGoneBranchesLocal(wd string) (*[]string, error) {
	// https://dev.heeus.io/launchpad/#!14544
	// 1. Step
	_, _, err := new(exec.PipedExec).
		Command(git, fetch, "-p", "--dry-run").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return nil, err
	}
	_, _, err = new(exec.PipedExec).
		Command(git, fetch, "-p").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return nil, err
	}
	// 2. Step
	stdout, _, err := new(exec.PipedExec).
		Command(git, branch, "-vv").
		WorkingDir(wd).
		Command("grep", "\\[[^]]*: gone[^]]*\\]").
		Command("gawk", "{print $1}").
		RunToStrings()
	if err != nil {
		return nil, err
	}
	// alternate: grep '\[.*: gone'
	mbrlocallist := strings.Split(stdout, caret)

	stsr := []string{}
	stdout, _, err = new(exec.PipedExec).
		Command(git, branch, "-r").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return nil, err
	}
	myremotelist := strings.Split(stdout, caret)
	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return nil, fmt.Errorf("failed to get main branch: %w", err)
	}

	curbr, err := GetCurrentBranchName(wd)
	if err != nil {
		return nil, err
	}

	for _, mylocalbranch := range mbrlocallist {
		mybranch := strings.TrimSpace(mylocalbranch)
		bfound := false
		if strings.Contains(mybranch, curbr) {
			bfound = true
		} else {
			if !strings.Contains(mybranch, mainBranch) && !strings.Contains(mybranch, "HEAD") {
				for _, mbranch := range myremotelist {
					mbranch = strings.TrimSpace(mbranch)
					if mybranch == mbranch {
						bfound = true
						break
					}
				}
			}
		}
		if !bfound {
			// delete branch in fork
			stsr = append(stsr, mybranch)
		}
	}
	return &stsr, nil
}

// DeleteBranchesLocal s.e.
func DeleteBranchesLocal(wd string, strs *[]string) error {
	for _, str := range *strs {
		if strings.TrimSpace(str) != "" {
			_, _, err := new(exec.PipedExec).
				Command(git, branch, "-D", str).
				WorkingDir(wd).
				RunToStrings()
			fmt.Printf("Branch %s deleted\n", str)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func GetNoteAndURL(notes []string) (note string, url string) {
	for _, s := range notes {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			if strings.Contains(s, httppref) {
				url = s
				if len(note) > 0 {
					break
				}
			} else {
				if note == "" {
					note = s
				} else {
					note = note + oneSpace + s
				}
				if strings.Contains(strings.ToLower(s), strings.ToLower(IssuePRTtilePrefix)) {
					break
				}
			}
		}
	}
	return note, url
}

func GetBodyFromNotes(notes []string) string {
	b := ""
	if (len(notes) > 1) && strings.Contains(strings.ToLower(notes[0]), strings.ToLower(IssuePRTtilePrefix)) {
		for i, note := range notes {
			note = strings.TrimSpace(note)
			if (strings.Contains(note, "https://") && !strings.Contains(note, "/issues/")) || !strings.Contains(note, "https://") {
				strings.Split(strings.ReplaceAll(note, "\r\n", caret), "")
				if i > 0 && len(note) > 0 {
					b += note
				}
			}
		}
	}
	return b
}

func createPR(wd, parentRepoName, prBranchName string, notes []string, asDraft bool) (stdout string, stderr string, err error) {
	if len(notes) == 0 {
		return "", "", errors.New(ErrMsgPRNotesImpossible)
	}

	//ParseGitRemoteURL()
	// get json notes object from dev branch
	notesObj, ok := notesPkg.Deserialize(notes)
	if !ok {
		return "", "", errors.New("error deserializing notes")
	}

	prTitle, err := getIssueDescription(notesObj.GithubIssueURL)
	if err != nil {
		return "", "", fmt.Errorf("Error retrieving issue description: %w", err)
	}

	var strNotes string
	var url string
	strNotes, url = GetNoteAndURL(notes)
	b := GetBodyFromNotes(notes)
	if len(b) == 0 {
		b = strNotes
	}
	if len(url) > 0 {
		b = b + caret + url
	}
	strBody := fmt.Sprintln(b)

	_, forkAccount, err := GetRepoAndOrgName(wd)
	if err != nil {
		return "", "", err
	}

	normalizedTitle := strings.ReplaceAll(prTitle, " ", "-")
	args := []string{
		"pr",
		"create",
		"--head",
		forkAccount + ":" + prBranchName,
		"--repo",
		parentRepoName,
		"--body",
		strBody,
		"--title",
		normalizedTitle,
	}
	if asDraft {
		args = append(args, "--draft")
	}
	err = helper.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command("gh", args...).
			RunToStrings()

		return err
	})
	if err != nil {
		return stdout, stderr, err
	}

	prExists, prURL, stdout, stderr, err := doesPrExist(wd, parentRepoName, prBranchName)
	if err != nil {
		return stdout, stderr, err
	}
	if !prExists {
		return stdout, stderr, errors.New("PR not created")
	}
	// print PR URL
	if len(prURL) > 0 {
		fmt.Println()
		fmt.Println(prURL)
	}

	return stdout, stderr, err
}

func retrieveRepoNameFromUPL(prurl string) string {
	var strs []string = strings.Split(prurl, slash)
	if len(strs) < minRepoNameLength {
		return ""
	}
	res := ""
	lenstr := len(strs)
	for i := lenstr - minRepoNameLength; i < lenstr-2; i++ {
		if res == "" {
			res = strs[i]
		} else {
			res = res + slash + strs[i]
		}
	}
	return res
}

func prCheckAbsent(val *gchResponse) bool {
	return strings.Contains(val._stderr, nochecksmsg)
}

func prCheckSuccess(val *gchResponse) bool {
	ss := strings.Split(val._stdout, caret)
	for _, s := range ss {
		if strings.Contains(s, "build") && strings.Contains(s, "pass") {
			return true
		}
	}
	return false
}

func waitPRChecks(parentrepo string, prurl string) *gchResponse {
	c := make(chan *gchResponse)

	// Run checking status of PR Checks
	go runPRChecksChecks(parentrepo, prurl, c)

	strw := msgWaitingPR
	var val *gchResponse
	var ok bool
	waitTimer := time.NewTimer(prTimeWait * time.Second)
	fmt.Print(strw)
	for {
		select {
		case val, ok = <-c:
			fmt.Println("")
			if ok {
				return val
			}
			return &gchResponse{_err: errors.New(ErrSomethigWrong)}
		case <-waitTimer.C:
			fmt.Println("")
			return &gchResponse{_err: errors.New(ErrTimer40Sec)}
		default:
			helper.Delay()
			fmt.Print(".")
		}
	}
}

func runPRChecksChecks(parentrepo string, prurl string, c chan *gchResponse) {
	var stdout, stderr string
	var err error
	if len(parentrepo) == 0 {
		stdout, stderr, err = new(exec.PipedExec).
			Command("gh", "pr", "checks", prurl, "--watch").
			RunToStrings()
	} else {
		stdout, stderr, err = new(exec.PipedExec).
			Command("gh", "pr", "checks", prurl, "--watch", "-R", parentrepo).
			RunToStrings()
	}
	c <- &gchResponse{stdout, stderr, err}
}

// getRemotes shows list of names of all remotes
func getRemotes(wd string) []string {
	stdout, _, _ := new(exec.PipedExec).
		Command(git, "remote").
		WorkingDir(wd).
		RunToStrings()
	strs := strings.Split(stdout, caret)
	for i, str := range strs {
		if len(strings.TrimSpace(str)) == 0 {
			strs = append(strs[:i], strs[i+1:]...)
		}
	}
	return strs
}

// GetFilesForCommit shows list of file names, ready for commit
func GetFilesForCommit(wd string) []string {
	stdout, _, _ := new(exec.PipedExec).
		Command(git, "status", "-s").
		WorkingDir(wd).
		RunToStrings()
	ss := strings.Split(stdout, caret)
	var strs []string
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			strs = append(strs, s)
		}
	}
	return strs
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

func setUpstreamBranch(wd string, repo string, branch string) error {
	mainBranchName, err := GetMainBranch(wd)
	if err != nil {
		return err
	}

	if branch == "" {
		branch = mainBranchName
	}

	return new(exec.PipedExec).
		Command(git, "push", "--set-upstream", repo, branch).
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
}

// GetCommitFileSizes returns quantity of cmmited files and their total sizes
func GetCommitFileSizes(wd string) (totalSize int, quantity int, err error) {
	totalSize = 0
	quantity = 0
	stdout, _, err := new(exec.PipedExec).
		Command(git, "status", "--porcelain").
		WorkingDir(wd).
		Command("gawk", "{if ($1 == \"??\") print $2}").
		RunToStrings()
	if err != nil {
		return 0, 0, err
	}
	files := strings.Split(stdout, caret)

	if len(files) == 0 {
		return
	}

	for _, file := range files {
		if len(file) > 0 {

			stdout, _, err = new(exec.PipedExec).
				Command("wc", "-c", file).
				Command("gawk", "{print $1}").
				RunToStrings()
			if err != nil {
				return 0, 0, err
			}

			strval := strings.TrimSpace(stdout)
			if strval != "" {
				sz, err := strconv.Atoi(strval)
				if err != nil {
					return 0, 0, fmt.Errorf("Error during conversion of value: %w", err)
				}
				totalSize += sz
				quantity++
			}
		}
	}
	return totalSize, quantity, nil
}

func getGlobalHookFolder() string {
	stdout, _, err := new(exec.PipedExec).
		Command(git, "config", "--global", "core.hooksPath").
		RunToStrings()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdout)
}

func getLocalHookFolder(wd string) (string, error) {
	dir, err := GetRootFolder(wd)
	if err != nil {
		return "", err
	}
	filename := "/.git/hooks/pre-commit"
	filepath := dir + filename

	return strings.TrimSpace(filepath), nil
}

// GlobalPreCommitHookExist - s.e.
func GlobalPreCommitHookExist() (bool, error) {
	filepath := getGlobalHookFolder()
	if len(filepath) == 0 {
		return false, nil // global hook folder not defined
	}
	err := os.MkdirAll(filepath, os.ModePerm)
	if err != nil {
		return false, err
	}

	filepath += "/pre-commit"
	// Check if the file already exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return false, nil // File pre-commit does not exist
	}

	return largeFileHookExist(filepath), nil
}

// LocalPreCommitHookExist - s.e.
func LocalPreCommitHookExist(wd string) (bool, error) {
	filepath, err := getLocalHookFolder(wd)
	if err != nil {
		return false, err
	}
	// Check if the file already exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return false, nil
	}

	return largeFileHookExist(filepath), nil
}

func largeFileHookExist(filepath string) bool {
	substring := "large-file-hook.sh"
	_, _, err := new(exec.PipedExec).Command("grep", "-l", substring, filepath).RunToStrings()

	return err == nil
}

// SetGlobalPreCommitHook - s.e.
func SetGlobalPreCommitHook(wd string) error {
	var err error
	path := getGlobalHookFolder()

	if len(path) == 0 {
		rootUser, err := user.Current()
		if err != nil {
			return err
		}

		path = rootUser.HomeDir
		path += "/.git/hooks"
		if err = os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
	}

	// Set global hooks folder
	err = new(exec.PipedExec).
		Command(git, "config", "--global", "core.hookspath", path).
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return err
	}

	filepath := path + "/pre-commit"
	f, err := createOrOpenFile(filepath)
	if err != nil {
		return err
	}

	_ = f.Close()
	if !largeFileHookExist(filepath) {
		return fillPreCommitFile(wd, filepath)
	}

	return nil
}

func GetRootFolder(wd string) (string, error) {
	stdout, _, err := new(exec.PipedExec).
		Command(git, "rev-parse", "--show-toplevel").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout), nil
}

// SetLocalPreCommitHook - s.e.
func SetLocalPreCommitHook(wd string) error {
	// Turn off globa1 hooks
	err := new(exec.PipedExec).
		Command(git, "config", "--global", "--get", "core.hookspath").
		Run(os.Stdout, os.Stdout)
	if err == nil {
		err = new(exec.PipedExec).
			Command(git, "config", "--global", "--unset", "core.hookspath").
			Run(os.Stdout, os.Stdout)
		if err != nil {
			return err
		}
	}
	dir, err := GetRootFolder(wd)
	if err != nil {
		return err
	}
	PreCommitHooksDirPath := filepath.Join(dir, ".git/hooks")

	if err := os.MkdirAll(PreCommitHooksDirPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	PreCommitFilePath := filepath.Join(PreCommitHooksDirPath, "pre-commit")

	// Check if the file already exists
	f, err := createOrOpenFile(PreCommitFilePath)
	if err != nil {
		return err
	}
	_ = f.Close()

	if !largeFileHookExist(PreCommitFilePath) {
		return fillPreCommitFile(wd, PreCommitFilePath)
	}

	return nil
}

func createOrOpenFile(filepath string) (*os.File, error) {
	_, err := os.Stat(filepath)
	var f *os.File
	if os.IsNotExist(err) {
		// Create file pre-commit
		f, err = os.Create(filepath)
		if err != nil {
			return nil, err
		}

		_, err = f.WriteString("#!/bin/bash\n")
	} else {
		f, err = os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, bashFilePerm)
	}
	if err != nil {
		return nil, err
	}

	return f, nil
}

func fillPreCommitFile(wd, myFilePath string) error {
	fPreCommit, err := createOrOpenFile(myFilePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = fPreCommit.Close()
	}()

	dir, err := GetRootFolder(wd)
	if err != nil {
		return err
	}
	fName := "/.git/hooks/large-file-hook.sh"
	lfPath := dir + fName

	lf, err := os.Create(lfPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = lf.Close()
	}()

	if _, err := lf.WriteString(largeFileHookContent); err != nil {
		return fmt.Errorf("failed to write large file hook content: %w", err)
	}

	preCommitContentBuf := strings.Builder{}
	preCommitContentBuf.WriteString("#!/bin/bash\n")
	preCommitContentBuf.WriteString("\n#Here is large files commit prevent is added by [qs]\n")
	preCommitContentBuf.WriteString("bash " + lfPath + caret)
	if _, err := fPreCommit.WriteString(preCommitContentBuf.String()); err != nil {
		return fmt.Errorf("failed to write pre-commit hook content: %w", err)
	}

	return new(exec.PipedExec).Command("chmod", "+x", myFilePath).Run(os.Stdout, os.Stdout)
}

func UpstreamNotExist(wd string) bool {
	return len(getRemotes(wd)) < 2
}

func HasRemote(wd, remoteName string) (bool, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "remote").
		WorkingDir(wd).
		RunToStrings()

	if err != nil {
		return false, fmt.Errorf("failed to list git remotes: %w: %s", err, stderr)
	}

	remotes := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, remote := range remotes {
		if strings.TrimSpace(remote) == remoteName {
			return true, nil
		}
	}

	return false, nil
}

func GawkInstalled() bool {
	_, _, err := new(exec.PipedExec).
		Command("gawk", "--version").
		RunToStrings()
	return err == nil
}

func extractIntegerPrefix(input string) (string, error) {
	// Define the regular expression pattern
	pattern := `^\d+`
	re := regexp.MustCompile(pattern)

	// Find the match
	match := re.FindString(input)
	if match == "" {
		return "", fmt.Errorf("no integer found at the beginning of the string")
	}

	// Convert the matched string to an integer
	integerValue, err := strconv.Atoi(match)
	if err != nil {
		return "", fmt.Errorf("error converting string to integer: %v", err)
	}

	return strconv.Itoa(integerValue), nil
}

func issuenumExists(parentrepo string, issuenum string) bool {
	stdouts, _, err := new(exec.PipedExec).
		Command("gh", "issue", "develop", issuenum, "--list", "-R", parentrepo).
		Command("gawk", "{print $2}").
		RunToStrings()
	if (err == nil) && (len(stdouts) > minIssueNoteLength) {
		names := strings.Split(stdouts, caret)
		for _, name := range names {
			if strings.Contains(name, slash+issuenum+"-") {
				return true
			}
		}
	}
	return false
}

func GetIssueNumFromBranchName(parentrepo string, curbranch string) (issuenum string, ok bool) {

	tempissuenum, err := extractIntegerPrefix(curbranch)
	if tempissuenum == "" {
		return "", false
	}
	if err == nil {
		if issuenumExists(parentrepo, tempissuenum) {
			return tempissuenum, true
		}
	}

	stdouts, stderr, err := new(exec.PipedExec).
		Command("gh", "issue", "list", "-R", parentrepo).
		Command("gawk", "{print $1}").
		RunToStrings()
	if err != nil {
		logger.Verbose("GetIssueNumFromBranchName:", stderr)
		return "", false
	}
	issuenums := strings.Split(stdouts, caret)
	fmt.Println("Searching linked issue ")

	for _, issuenum := range issuenums {
		if len(issuenum) > 0 {
			fmt.Println("  Issue number: ", issuenum, "...")
			if issuenumExists(parentrepo, issuenum) {
				return issuenum, true
			}
		}
	}

	return "", false
}

func GetIssuePRTitle(issueNum string, parentrepo string) []string {
	name := GetIssueNameByNumber(issueNum, parentrepo)
	s := IssuePRTtilePrefix + oneSpace + name
	body := IssueSign + issueNum + oneSpace + name
	return []string{s, body}
}

func LinkIssueToMileStone(issueNum string, parentrepo string) error {
	if issueNum == "" {
		return nil
	}
	if parentrepo == "" {
		return nil
	}
	strMilestones, _, err := new(exec.PipedExec).
		Command("gh", "api", "repos/"+parentrepo+"/milestones", "--jq", ".[] | .title").
		RunToStrings()
	if err != nil {
		return fmt.Errorf("Link issue to mileStone error: %w", err)
	}
	milestones := strings.Split(strMilestones, caret)
	// Sample date string in the "yyyy.mm.dd" format.
	dateString := "2006.01.02"
	// Get the current date and time.
	currentTime := time.Now()
	for _, milestone := range milestones {
		// Parse the input string into a time.Time value.
		t, err := time.Parse(dateString, milestone)
		if err == nil {
			if currentTime.Before(t) {
				// Next milestone is found
				err = new(exec.PipedExec).
					Command("gh", "issue", "edit", issueNum, "--milestone", milestone, "--repo", parentrepo).
					Run(os.Stdout, os.Stdout)
				if err != nil {
					return err
				}
				fmt.Println("Issue #" + issueNum + " added to milestone '" + milestone + "'")
				return nil
			}
		}
	}

	return nil
}

func GetCurrentBranchName(wd string) (string, error) {
	branchName, _, err := new(exec.PipedExec).
		Command(git, branch, "--show-current").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch name: %v", err)
	}

	return strings.TrimSpace(branchName), nil
}

// createRemote creates a remote in the cloned repository
func CreateRemote(wd, remote, account, token, repoName string, isUpstream bool) error {
	repo, err := goGitPkg.PlainOpen(wd)
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
	}

	if err = repo.DeleteRemote(remote); err != nil {
		if !errors.Is(err, goGitPkg.ErrRemoteNotFound) {
			return fmt.Errorf("failed to delete %s remote: %w", remote, err)
		}
	}

	remoteURL := BuildRemoteURL(account, token, repoName, isUpstream)
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: remote,
		URLs: []string{remoteURL},
	})
	if err != nil {
		return fmt.Errorf("failed to create %s remote: %w", remote, err)
	}

	return nil
}

// buildRemoteURL constructs the remote URL for cloning
func BuildRemoteURL(account, token, repoName string, isUpstream bool) string {
	return "https://" + account + ":" + token + "@github.com/" + account + "/" + repoName + ".git"
}

// IamInMainBranch checks if current branch is main branch
// Returns:
// - the name of current branch
// - true if current branch is main branch
// - error if any
func IamInMainBranch(wd string) (string, bool, error) {
	currentBranchName, err := GetCurrentBranchName(wd)
	if err != nil {
		return "", false, err
	}
	logger.Verbose("Current branch: " + currentBranchName)

	mainBranch, err := GetMainBranch(wd)
	logger.Verbose("Main branch: " + mainBranch)
	if err != nil {
		return "", false, fmt.Errorf("failed to get main branch: %w", err)
	}

	return currentBranchName, strings.EqualFold(currentBranchName, mainBranch), err
}

// RemoveRepo removes a repository from GitHub
func RemoveRepo(repoName, account, token string) error {
	cmd := osExec.Command("gh", "repo", "delete",
		fmt.Sprintf("%s/%s", account, repoName),
		"--yes")

	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GITHUB_TOKEN=%s", token))

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete repository: %w\nOutput: %s", err, output)
	}

	return nil
}

// GetRemoteUrlByName retrieves the URL of a specified remote by its name
func GetRemoteUrlByName(wd string, remoteName string) (string, error) {
	repo, err := goGitPkg.PlainOpen(wd)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	remotes, err := repo.Remotes()
	if err != nil {
		return "", fmt.Errorf("failed to get remotes: %w", err)
	}

	for _, remote := range remotes {
		if remote.Config().Name == remoteName {
			if len(remote.Config().URLs) > 0 {
				return remote.Config().URLs[0], nil
			}
			return "", fmt.Errorf("remote %s has no URLs configured", remoteName)
		}
	}

	return "", fmt.Errorf("remote %s not found", remoteName)
}

// CleanupTestEnvironment removes all created resources: cloned repo, upstream repo, fork repo
func CleanupTestEnvironment(cloneRepoPath, anotherCloneRepoPath string) error {
	// Remove the another cloned repository
	if anotherCloneRepoPath != "" {
		if err := os.RemoveAll(anotherCloneRepoPath); err != nil {
			return fmt.Errorf("failed to remove another cloned repository: %w", err)
		}
	}

	// get remotes from main clone
	remoteOriginURL, err := GetRemoteUrlByName(cloneRepoPath, "origin")
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			remoteOriginURL = ""
		} else {
			return fmt.Errorf("failed to get oririn remote URL: %w", err)
		}
	}

	remoteUpstreamURL, err := GetRemoteUrlByName(cloneRepoPath, "upstream")
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			remoteUpstreamURL = ""
		} else {
			return fmt.Errorf("failed to get upstream remote URL: %w", err)
		}
	}

	// Remove the cloned repository
	if err := os.RemoveAll(cloneRepoPath); err != nil {
		return fmt.Errorf("failed to remove cloned repository: %w", err)
	}

	var errList []error
	// extract account, repo and token from remote url
	forkAccount, repo, forkToken, err := ParseGitRemoteURL(remoteOriginURL)
	if err == nil {
		// Optionally remove the fork repository
		if err := RemoveRepo(repo, forkAccount, forkToken); err != nil {
			errList = append(errList, errors.Join(fmt.Errorf("failed to remove origin repository: %w", err)))
		}
	}

	// extract account, repo and token from remote url
	upstreamAccount, repo, upstreamToken, err := ParseGitRemoteURL(remoteUpstreamURL)
	if err == nil {
		// Optionally remove the upstream repository
		if err := RemoveRepo(repo, upstreamAccount, upstreamToken); err != nil {
			errList = append(errList, errors.Join(fmt.Errorf("failed to remove upstream repository: %w", err)))
		}
	}

	if len(errList) > 0 {
		return errors.Join(errList...)
	}

	return nil
}
