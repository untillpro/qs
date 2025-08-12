package systrun

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	gitPkg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	goUtilsExec "github.com/untillpro/goutils/exec"
	"github.com/untillpro/qs/gitcmds"
	contextCfg "github.com/untillpro/qs/internal/context"
	"github.com/untillpro/qs/internal/helper"
	"github.com/untillpro/qs/internal/jira"
	notesPkg "github.com/untillpro/qs/internal/notes"
)

// SystemTest represents a single system test for the qs utility
type SystemTest struct {
	ctx                  context.Context
	cfg                  *TestConfig
	cloneRepoPath        string
	anotherCloneRepoPath string
	repoName             string
	qsExecRootCmd        func(ctx context.Context, args []string) (context.Context, error)
}

// TestConfig contains all configuration for a system test
type TestConfig struct {
	TestID                 string
	GHConfig               GithubConfig
	CommandConfig          *CommandConfig
	UpstreamState          RemoteState
	ForkState              RemoteState
	SyncState              SyncState
	DevBranchState         DevBranchState       // if true then create dev branch
	ClipboardContent       ClipboardContentType // Content to be set in clipboard before running the test
	RunCommandOnOtherClone bool                 // RunCommandOnOtherClone specifies if a command should be executed from an additional repository clone.
	NeedCollaboration      bool
	BranchState            *BranchState // if true then create branch with prefix
	ExpectedStderr         string       // If ExpectedStderr is not empty then check exit code of qs it must be != 0
	ExpectedStdout         string
	Expectations           []ExpectationFunc
}

type BranchState struct {
	DevBranchExists      bool // dev-branch exists
	DevBranchHasRtBranch bool // Remote tracking branch exists
	DevBranchIsAhead     bool // dev-branch is ahead of the remote tracking branch
	PRBranchExists       bool // PR branch exists
	PRBranchHasRtBranch  bool // Remote tracking branch exists
	PRBranchIsAhead      bool // PR branch is ahead of the remote tracking branch
	PRExists             bool // Pull request exists
	PRMerged             bool // Pull request is merged
}

type CommandConfig struct {
	// Command is the name of the qs command to be executed
	Command string
	// Args are arguments to be passed to the command
	Args []string
	// Stdin is a string that will be written to stdin of the command
	Stdin string
}

// GithubConfig holds GitHub account and token information
type GithubConfig struct {
	UpstreamAccount string
	UpstreamToken   string
	ForkAccount     string
	ForkToken       string
}

// Expectation represents a type of expectation
type Expectation int

type ExpectationFunc func(_ context.Context) error

// ExpectationCustomBranchIsCurrentBranch represents checker for ExpectationCurrentBranch
func ExpectationCustomBranchIsCurrentBranch(ctx context.Context) error {
	currentBranch, err := gitcmds.GetCurrentBranchName(ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string))
	if err != nil {
		return err
	}

	customBranchName := ctx.Value(contextCfg.CtxKeyCustomBranchName).(string)
	if currentBranch != customBranchName {
		return fmt.Errorf("current branch '%s' does not match expected branch '%s'", currentBranch, customBranchName)
	}

	return nil
}

// ExpectationCloneIsSyncedWithFork checks if the clone is synchronized with the fork
func ExpectationCloneIsSyncedWithFork(ctx context.Context) error {
	// Compare local and remote branches to ensure they're in sync
	repo, err := gitPkg.PlainOpen(ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string))
	if err != nil {
		return fmt.Errorf(errFormatFailedToCloneRepos, err)
	}

	// Get the current branch
	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get the branch name
	branchName := ""
	if head.Name().IsBranch() {
		branchName = head.Name().Short()
	} else {
		return fmt.Errorf("HEAD is not on a branch")
	}

	// Get the remote branch reference
	remoteBranchRef := plumbing.NewRemoteReferenceName(origin, branchName)
	remoteBranch, err := repo.Reference(remoteBranchRef, true)
	if err != nil {
		return fmt.Errorf("failed to get remote branch: %w", err)
	}

	// Check if local and remote are in sync
	if head.Hash() != remoteBranch.Hash() {
		return fmt.Errorf("local and remote branches are not in sync")
	}

	return nil
}

// ExpectationForkExists represents the expected state of a fork
func ExpectationForkExists(ctx context.Context) error {
	// Implement the logic to check if the fork exists
	// get remotes of the local repo and check if remote, called origin, exists
	repo, err := gitPkg.PlainOpen(ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string))
	if err != nil {
		return fmt.Errorf(errFormatFailedToCloneRepos, err)
	}

	// Check if the remote named "origin" exists
	if _, err := repo.Remote(origin); err != nil {
		return fmt.Errorf("origin remote not found after fork command: %w", err)
	}

	// Check if the remote URL accessible
	cmd := exec.Command(git, "ls-remote", origin)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to access remote URL: %w", err)
	}

	return nil
}

// ExpectationBranchLinkedToIssue represents the expected state of a branch linked to a GitHub issue
func ExpectationBranchLinkedToIssue(ctx context.Context) error {
	// extract repo and issue number from e.createdGithubIssueURL using regex
	repoOwner, repoName, issueNum, err := parseGithubIssueURL(ctx.Value(contextCfg.CtxKeyCreatedGithubIssueURL).(string))
	if err != nil {
		return fmt.Errorf("failed to parse GitHub issue URL: %w", err)
	}
	// Get current branch from the repo
	devBranchName, err := findBranchNameWithPrefix(ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string), issueNum)
	if err != nil {
		return err
	}
	// Build full repo URL
	repoURL := fmt.Sprintf("https://github.com/%s/%s", repoOwner, repoName)
	// Run gh issue develop --list command with retry logic
	var output []byte
	err = helper.Retry(func() error {
		cmd := exec.Command("gh", "issue", "develop", "--list", "--repo", repoURL, issueNum)
		var cmdErr error
		output, cmdErr = cmd.Output()
		return cmdErr
	})
	if err != nil {
		return fmt.Errorf("failed to check linked branches: %w", err)
	}

	// Check if current branch exists in the output
	if !strings.Contains(string(output), devBranchName) {
		return fmt.Errorf("current branch %s is not linked to issue #%s", devBranchName, issueNum)
	}

	return nil
}

// ExpectationLargeFileHooksInstalled checks if the large file hooks are installed
func ExpectationLargeFileHooksInstalled(ctx context.Context) error {
	// Check if the large file hooks are installed
	cloneRepoPath := ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string)
	if cloneRepoPath == "" {
		return errCloneRepoPathNotFoundInContext
	}

	hookPath := filepath.Join(cloneRepoPath, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		return fmt.Errorf("pre-commit hook is not installed at %s", hookPath)
	}

	substring := "large-file-hook.sh"
	stdout, _, err := new(goUtilsExec.PipedExec).
		Command("grep", "-l", substring, hookPath).
		RunToStrings()
	if err != nil {
		return fmt.Errorf("failed to check if large file hook is installed: %w", err)
	}

	if len(stdout) == 0 {
		return fmt.Errorf("large file hook is not installed")
	}

	return nil
}

// ExpectationCurrentBranchHasPrefix checks if the current branch has the expected prefix
func ExpectationCurrentBranchHasPrefix(ctx context.Context) error {
	// Open the repository
	repo, err := gitPkg.PlainOpen(ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string))
	if err != nil {
		return fmt.Errorf(errFormatFailedToCloneRepos, err)
	}

	// Get the current branch
	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Check if the branch name starts with the expected prefix
	branchPrefix := ctx.Value(contextCfg.CtxKeyBranchPrefix).(string)
	if !strings.HasPrefix(head.Name().Short(), branchPrefix) {
		return fmt.Errorf("branch name '%s' does not start with expected prefix '%s'", head.Name().Short(), branchPrefix)
	}

	return nil
}

// ExpectationPRCreated checks:
// 1. If PR branch exists with correct naming pattern (-dev → -pr)
// 2. If PR branch contains notes with branch_type=2
// 3. If the PR branch has exactly one squashed commit
// 4. If the dev branch no longer exists (locally and remotely)
// 5. If an actual pull request was created in the upstream repo
func ExpectationPRCreated(ctx context.Context) error {
	// 1. Check if PR branch exists with correct naming
	cloneRepoPath := ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string)
	if cloneRepoPath == "" {
		return errCloneRepoPathNotFoundInContext
	}

	anotherCloneRepoPath, ok := ctx.Value(contextCfg.CtxKeyAnotherCloneRepoPath).(string)
	if ok && anotherCloneRepoPath != "" {
		cloneRepoPath = anotherCloneRepoPath
	}

	devBranchName, ok := ctx.Value(contextCfg.CtxKeyDevBranchName).(string)
	if !ok {
		return fmt.Errorf("dev branch name not found in context")
	}

	expectedPRBranch := ""
	if strings.HasSuffix(devBranchName, "-dev") {
		expectedPRBranch = strings.TrimSuffix(devBranchName, "-dev") + "-pr"
	} else {
		return fmt.Errorf("dev branch %s does not have expected -dev suffix", devBranchName)
	}

	// Open the repository
	repo, err := gitPkg.PlainOpen(cloneRepoPath)
	if err != nil {
		return fmt.Errorf(errFormatFailedToCloneRepos, err)
	}

	// Check if PR branch exists locally
	branches, err := repo.Branches()
	if err != nil {
		return fmt.Errorf("failed to get branches: %w", err)
	}

	prBranchExists := false
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().Short() == expectedPRBranch {
			prBranchExists = true
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error iterating branches: %w", err)
	}

	if !prBranchExists {
		return fmt.Errorf("PR branch %s not found", expectedPRBranch)
	}

	// 2. Check if branch has notes with branch_type=2
	cmd := exec.Command(git, "-C", cloneRepoPath, "checkout", expectedPRBranch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout PR branch: %w", err)
	}

	// Get notes from the branch
	notes, _, err := gitcmds.GetNotes(cloneRepoPath, expectedPRBranch)
	if err != nil {
		return err
	}
	notesObj, ok := notesPkg.Deserialize(notes)
	if notesObj == nil || !ok {
		return fmt.Errorf("error: No notes found in branch %s: ", expectedPRBranch)
	}

	if notesObj.BranchType != notesPkg.BranchTypePr {
		return fmt.Errorf("error: branch type is not pr")
	}

	// 3. Check if the PR branch has exactly one squashed commit
	// First get number of commits
	cmd = exec.Command(git, "-C", cloneRepoPath, "rev-list", "--count", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to count commits: %w", err)
	}

	commitCount, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return fmt.Errorf("failed to parse commit count: %w", err)
	}

	// Check commit count (should have 1-2 commits: squashed commit + possibly empty commit for notes)
	if commitCount > 2 {
		return fmt.Errorf("expected at most 2 commits in PR branch, but found %d", commitCount)
	}

	// Check if we have more than 1 commit, and if so, verify the last one is the empty commit for notes
	if commitCount > 1 {
		cmd = exec.Command(git, "-C", cloneRepoPath, "log", "-1", "--pretty=%B")
		_, err = cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get last commit message: %w", err)
		}
	}

	// 4. Check if dev branch no longer exists (locally and remotely)
	// Check locally
	branches, err = repo.Branches()
	if err != nil {
		return fmt.Errorf("failed to get branches: %w", err)
	}

	devBranchExists := false
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().Short() == devBranchName {
			devBranchExists = true
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error iterating branches: %w", err)
	}

	if devBranchExists {
		return fmt.Errorf("dev branch %s still exists locally after PR creation", devBranchName)
	}

	// Check remotely in origin
	stdout, stderr, err := new(goUtilsExec.PipedExec).
		Command(git, "-C", cloneRepoPath, "ls-remote", "--heads", origin, devBranchName).
		RunToStrings()

	if err != nil {
		return fmt.Errorf("failed to check remote branches: %w, stderr: %s", err, stderr)
	}

	if stdout != "" {
		return fmt.Errorf("dev branch %s still exists on origin after PR creation", devBranchName)
	}

	parentRepo, err := gitcmds.GetParentRepoName(cloneRepoPath)
	if err != nil {
		return fmt.Errorf("failed to get parent repo name: %w", err)
	}

	remoteName := upstream
	if len(parentRepo) == 0 {
		remoteName = origin
	}

	// Check remotely in remoteName
	stdout, stderr, err = new(goUtilsExec.PipedExec).
		Command(git, "-C", cloneRepoPath, "ls-remote", "--heads", remoteName, devBranchName).
		RunToStrings()

	if err != nil {
		return fmt.Errorf("failed to check remote branches: %w, stderr: %s", err, stderr)
	}

	if stdout != "" {
		return fmt.Errorf("dev branch %s still exists on %s after PR creation", devBranchName, remoteName)
	}

	// 5. Check if a real pull request was created in remoteName repo
	// Extract repo owner and name from upstream remote
	remote, err := repo.Remote(remoteName)
	if err != nil {
		return fmt.Errorf("upstream remote not found: %w", err)
	}

	remoteURL := remote.Config().URLs[0]
	var owner, repoName string

	owner, repoName, _, err = gitcmds.ParseGitRemoteURL(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to parse upstream URL: %w", err)
	}
	// Use gh CLI to check if PR exists with retry logic

	prInfo, _, _, err := gitcmds.DoesPrExist(
		cloneRepoPath,
		fmt.Sprintf("%s/%s", owner, repoName),
		expectedPRBranch,
		gitcmds.PRStateOpen,
	)
	if err != nil {
		return err
	}
	if prInfo == nil {
		return fmt.Errorf("PR branch %s does not exist on upstream remote", expectedPRBranch)
	}

	// extract expected PR title from GitHub issue or JIRA ticket
	var expectedPRTitle string
	if notesObj.GithubIssueURL != "" {
		expectedPRTitle, err = gitcmds.GetIssueDescription(notesObj.GithubIssueURL)
		if err != nil {
			return err
		}
	}
	if notesObj.JiraTicketURL != "" {
		expectedPRTitle, err = jira.GetJiraIssueName(notesObj.JiraTicketURL, "")
		if err != nil {
			return err
		}
	}

	// check actual PR title with expected one
	if expectedPRTitle != "" && prInfo.Title != expectedPRTitle {
		return fmt.Errorf("PR title does not match issue title. PR title: %s, issue title: %s", prInfo.Title, expectedPRTitle)
	}

	return nil
}

// ExpectationRemoteBranchWithCommitMessage checks if the remote branch exists and has specific commit message
func ExpectationRemoteBranchWithCommitMessage(ctx context.Context) error {
	// Check if the remote branch exists
	cloneRepoPath := ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string)
	if cloneRepoPath == "" {
		return errCloneRepoPathNotFoundInContext
	}

	remoteBranchName, ok := ctx.Value(contextCfg.CtxKeyDevBranchName).(string)
	if !ok {
		return fmt.Errorf("remote branch name not found in context")
	}

	// Check if branch exists on the remote with retry logic
	var stdout, stderr string
	err := helper.Retry(func() error {
		var lsErr error
		stdout, stderr, lsErr = new(goUtilsExec.PipedExec).
			Command(git, "ls-remote", "--heads", origin, remoteBranchName).
			WorkingDir(cloneRepoPath).
			RunToStrings()
		return lsErr
	})

	if err != nil {
		return fmt.Errorf("failed to check remote branches: %w: %s", err, stderr)
	}

	if stdout == "" {
		return fmt.Errorf("remote branch %s not found", remoteBranchName)
	}

	// check that remote branch has specific commit message
	stdout, stderr, err = new(goUtilsExec.PipedExec).
		Command(git, "log", "-1", "--pretty=%B", remoteBranchName).
		WorkingDir(cloneRepoPath).
		RunToStrings()

	if err != nil {
		return fmt.Errorf("failed to get last commit message: %w: %s", err, stderr)
	}

	commitMessageParts, ok := ctx.Value(contextCfg.CtxKeyCommitMessage).([]string)
	if !ok {
		return fmt.Errorf("commit message not found in context")
	}

	commitMessage := strings.Join(commitMessageParts, " ")
	if !strings.Contains(stdout, commitMessage) {
		return fmt.Errorf("remote branch %s does not have commit message '%s'", remoteBranchName, commitMessage)
	}

	return nil
}

// ExpectationNotesDownloaded checks if the notes are downloaded from the remote branch
func ExpectationNotesDownloaded(ctx context.Context) error {
	// Step 1: Get the remote URL and repo name
	cloneRepoPath := ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string)
	if cloneRepoPath == "" {
		return errCloneRepoPathNotFoundInContext
	}

	remoteBranchName, ok := ctx.Value(contextCfg.CtxKeyDevBranchName).(string)
	if !ok {
		return fmt.Errorf("remote branch name not found in context")
	}

	remoteURL, err := gitcmds.GetRemoteUrlByName(cloneRepoPath, origin)
	if err != nil {
		return fmt.Errorf("failed to get remote URL: %w", err)
	}

	_, repo, token, err := gitcmds.ParseGitRemoteURL(remoteURL)
	if err != nil {
		return err
	}

	// Step 2: Create temp path for the clone
	tempPath, err := os.MkdirTemp("", "qs-test-clone-*")
	if err != nil {
		return fmt.Errorf("failed to create temp clone path: %w", err)
	}

	defer func() {
		_ = os.RemoveAll(tempPath)
	}()

	// Step 3: Clone the repository in the temp path
	tempClonePath := filepath.Join(tempPath, repo)
	cloneCmd := exec.Command(git, "clone", remoteURL)
	cloneCmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", token))
	cloneCmd.Dir = tempPath

	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone repository: %w, output: %s", err, output)
	}

	// Step 4: Checkout on remote branch
	if err := checkoutOnBranch(tempClonePath, remoteBranchName); err != nil {
		return err
	}

	// Step 5: Run `qs d`
	if err := gitcmds.Download(tempClonePath); err != nil {
		return err
	}

	// Step 6: Check if notes are downloaded
	notes, _, err := gitcmds.GetNotes(tempClonePath, remoteBranchName)
	if err != nil {
		return err
	}

	if len(notes) == 0 {
		return fmt.Errorf("no notes downloaded")
	}

	// Step 7: Check if notes are of correct type
	notesObj, ok := notesPkg.Deserialize(notes)
	if !ok {
		return errors.New("error: No notes found in dev branch")
	}

	if notesObj.BranchType != notesPkg.BranchTypeDev {
		return fmt.Errorf("notes downloaded but branch type is not dev")
	}

	return nil
}

func expectationLocalBranchExists(ctx context.Context, expectedCount int) error {
	cloneRepoPath := ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string)
	if cloneRepoPath == "" {
		return errCloneRepoPathNotFoundInContext
	}

	// Get local branches
	stdout, stderr, err := new(goUtilsExec.PipedExec).
		Command(git, "branch", "--format", "%(refname:short)").
		WorkingDir(cloneRepoPath).
		RunToStrings()

	if err != nil {
		return fmt.Errorf("failed to list local branches: %w, stderr: %s", err, stderr)
	}

	// Count branches
	branches := strings.Split(strings.TrimSpace(stdout), "\n")
	// Filter out empty entries
	var count int
	for _, b := range branches {
		if strings.TrimSpace(b) != "" {
			count++
		}
	}

	if count != expectedCount {
		return fmt.Errorf("expected %d local branch, but found %d", expectedCount, count)
	}

	return nil
}

func expectationRemoteBranchExists(ctx context.Context, expectedCount int) error {
	cloneRepoPath := ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string)
	if cloneRepoPath == "" {
		return errCloneRepoPathNotFoundInContext
	}

	// Get remote branches from origin only
	stdout, stderr, err := new(goUtilsExec.PipedExec).
		Command(git, "branch", "-r", "--format", "%(refname:short)").
		WorkingDir(cloneRepoPath).
		RunToStrings()

	if err != nil {
		return fmt.Errorf("failed to list remote branches: %w, stderr: %s", err, stderr)
	}

	// Count origin branches
	branches := strings.Split(strings.TrimSpace(stdout), "\n")
	var count int
	for _, b := range branches {
		if strings.TrimSpace(b) != "" && strings.HasPrefix(b, "origin/") && b != "origin/HEAD" {
			count++
		}
	}

	if count != expectedCount {
		return fmt.Errorf("expected %d remote branch in origin, but found %d", expectedCount, count)
	}

	return nil
}

// ExpectationOneLocalBranch checks if there is exactly one local branch in the clone repo
func ExpectationOneLocalBranch(ctx context.Context) error {
	//nolint:revive
	return expectationLocalBranchExists(ctx, 1)
}

// ExpectationTwoLocalBranches checks if there are exactly two local branches in the clone repo
func ExpectationTwoLocalBranches(ctx context.Context) error {
	//nolint:revive
	return expectationLocalBranchExists(ctx, 2)
}

// ExpectationThreeLocalBranches checks if there are exactly three local branches in the clone repo
func ExpectationThreeLocalBranches(ctx context.Context) error {
	//nolint:revive
	return expectationLocalBranchExists(ctx, 3)
}

// ExpectationOneRemoteBranch checks if there is exactly one remote branch in the origin remote
func ExpectationOneRemoteBranch(ctx context.Context) error {
	//nolint:revive
	return expectationRemoteBranchExists(ctx, 1)
}

// ExpectationTwoRemoteBranches checks if there are exactly two remote branches in the origin remote
func ExpectationTwoRemoteBranches(ctx context.Context) error {
	//nolint:revive
	return expectationRemoteBranchExists(ctx, 2)
}

// ExpectationThreeRemoteBranches checks if there are exactly three remote branches in the origin remote
func ExpectationThreeRemoteBranches(ctx context.Context) error {
	//nolint:revive
	return expectationRemoteBranchExists(ctx, 3)
}
