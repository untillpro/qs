package systrun

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing/object"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	goUtilsExec "github.com/untillpro/goutils/exec"
	gitCmds "github.com/untillpro/qs/gitcmds"
	contextCfg "github.com/untillpro/qs/internal/context"
)

// SystemTest represents a single system test for the qs utility
type SystemTest struct {
	ctx           context.Context
	cfg           *TestConfig
	cloneRepoPath string
	repoName      string
	qsExecRootCmd func(ctx context.Context, args []string) (context.Context, error)
}

// TestConfig contains all configuration for a system test
type TestConfig struct {
	TestID        string
	GHConfig      GithubConfig
	CommandConfig CommandConfig
	UpstreamState RemoteState
	ForkState     RemoteState
	// TODO: if not 0 then run `qs dev` command, make modifications in files and run `qs u` command and then implement specified sync state
	// e.g. if SyncState is SyncStateSynchronized then do nothing more
	// e.g. if SyncStateForkChanged then additionally one push from another clone
	SyncState        SyncState
	DevBranchState   DevBranchState       // if true then create dev branch
	ClipboardContent ClipboardContentType // Content to be set in clipboard before running the test
	// If ExpectedStderr is not empty then check exit code of qs it must be != 0
	ExpectedStderr string
	ExpectedStdout string
	Expectations   []IExpectation
}

type CommandConfig struct {
	// Command is the name of the qs command to be executed
	Command string
	// Args are arguments to be passed to the command
	Args []string
	// Stdin is a string that will be written to stdin of the command
	Stdin string
}

type RuntimeEnvironment struct {
	// URL of the created GitHub issue
	createdGithubIssueURL string
	// Path to the cloned repository
	cloneRepoPath string
	// Custom name of the branch created during the test
	customBranchName string
	// Prefix for the branch name, used to check if the branch is created correctly
	branchPrefix  string
	devBranchName string
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

const (
	ExpectationCurrentBranch Expectation = iota
	ExpectationRemoteState
	ExpectationPullRequest
	ExpectationDownloadResult
	ExpectationUploadResult
	ExpectationForkExists
	ExpectationBranchLinkedToIssue
	ExpectationCurrentBranchHasPrefix
	ExpectationPRBranchState
	ExpectationCommitsFromAnotherClone
	ExpectationRemoteBranch
)

// Available expectations
// Each Expectation type has its own struct that implements IExpectation interface
var availableExpectations = map[Expectation]IExpectation{
	ExpectationCurrentBranch:           expectedCurrentBranch{},
	ExpectationRemoteState:             expectedRemoteState{},
	ExpectationPullRequest:             expectedPullRequest{},
	ExpectationDownloadResult:          expectedDownloadResult{},
	ExpectationUploadResult:            expectedUploadResult{},
	ExpectationForkExists:              expectedForkExists{},
	ExpectationBranchLinkedToIssue:     expectedBranchLinkedToIssue{},
	ExpectationPRBranchState:           expectedPRBranchState{},
	ExpectationCurrentBranchHasPrefix:  expectedCurrentBranchHasPrefix{},
	ExpectationCommitsFromAnotherClone: expectedCommitsFromAnotherClone{},
	ExpectationRemoteBranch:            expectedRemoteBranch{},
}

// Expectations returns a list of expectations based on the provided types
func Expectations(types ...Expectation) []IExpectation {
	expectations := make([]IExpectation, 0, len(types))
	for _, t := range types {
		iExpectation, ok := availableExpectations[t]
		if !ok {
			panic(fmt.Sprintf("unknown expectation type: %v", t))
		}

		expectations = append(expectations, iExpectation)
	}

	return expectations
}

// expectedCurrentBranch represents checker for ExpectationCurrentBranch
type expectedCurrentBranch struct{}

func (e expectedCurrentBranch) Check(ctx context.Context) error {
	currentBranch := gitCmds.GetCurrentBranchName(ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string))
	customBranchName := ctx.Value(contextCfg.CtxKeyCustomBranchName).(string)
	if currentBranch != customBranchName {
		return fmt.Errorf("current branch '%s' does not match expected branch '%s'", currentBranch, customBranchName)
	}

	return nil
}

// ExpectedRemoteState represents the expected state of a remote
type expectedRemoteState struct{}

func (e expectedRemoteState) Check(_ context.Context) error {
	// Implement the logic to check the remote state

	return nil
}

// ExpectedPullRequest represents the expected state of a pull request
type expectedPullRequest struct {
	Exists          bool
	Title           string
	ForkAccount     string
	UpstreamAccount string
}

func (e expectedPullRequest) Check(ctx context.Context) error {
	// Open the repository
	repo, err := git.PlainOpen(ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string))
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
	}

	// Check remotes configuration
	origin, err := repo.Remote("origin")
	if err != nil {
		return fmt.Errorf("origin remote not found after fork command: %w", err)
	}

	// Verify origin points to fork
	expectedForkURL := fmt.Sprintf("https://github.com/%s/", e.ForkAccount)
	if !strings.Contains(origin.Config().URLs[0], expectedForkURL) {
		return fmt.Errorf("origin remote does not point to fork: %s", origin.Config().URLs[0])
	}

	// Verify upstream remote exists and points to upstream
	upstream, err := repo.Remote("upstream")
	if err != nil {
		return fmt.Errorf("upstream remote not found after fork command: %w", err)
	}

	expectedUpstreamURL := fmt.Sprintf("https://github.com/%s/", e.UpstreamAccount)
	if !strings.Contains(upstream.Config().URLs[0], expectedUpstreamURL) {
		return fmt.Errorf("upstream remote does not point to upstream: %s", upstream.Config().URLs[0])
	}

	return nil
}

// ExpectedDownloadResult represents the expected state after downloading changes
type expectedDownloadResult struct{}

func (e expectedDownloadResult) Check(ctx context.Context) error {
	// Compare local and remote branches to ensure they're in sync
	repo, err := git.PlainOpen(ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string))
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
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
	remoteBranchRef := plumbing.NewRemoteReferenceName("origin", branchName)
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

// ExpectedUploadResult represents the expected state after uploading changes
type expectedUploadResult struct{}

func (e expectedUploadResult) Check(ctx context.Context) error {
	// Similar to download but checks if the changes were pushed to remote
	repo, err := git.PlainOpen(ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string))
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
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

	// Execute git fetch to get latest remote state
	cmd := exec.Command("git", "fetch", "origin")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch from remote: %w", err)
	}

	// Check if local branch is ahead of remote branch
	cmd = exec.Command("git", "rev-list", "--count",
		fmt.Sprintf("origin/%s..%s", branchName, branchName))
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check if branch is ahead: %w", err)
	}

	aheadCount, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return fmt.Errorf("failed to parse ahead count: %w", err)
	}

	if aheadCount > 0 {
		return fmt.Errorf("local branch is ahead of remote by %d commits", aheadCount)
	}

	return nil
}

// ExpectedForkExists represents the expected state of a fork
type expectedForkExists struct{}

func (e expectedForkExists) Check(ctx context.Context) error {
	// Implement the logic to check if the fork exists
	// get remotes of the local repo and check if remote, called origin, exists
	repo, err := git.PlainOpen(ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string))
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
	}

	// Check if the remote named "origin" exists
	if _, err := repo.Remote("origin"); err != nil {
		return fmt.Errorf("origixn remote not found after fork command: %w", err)
	}

	// Check if the remote URL accessible
	cmd := exec.Command("git", "ls-remote", "origin")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to access remote URL: %w", err)
	}

	return nil
}

// expectedBranchLinkedToIssue represents the expected state of a branch linked to a GitHub issue
type expectedBranchLinkedToIssue struct{}

// Check verifies if the branch is linked to a GitHub issue
func (e expectedBranchLinkedToIssue) Check(ctx context.Context) error {
	// extract repo and issue number from e.createdGithubIssueURL using regex
	repoOwner, repoName, issueNum, err := parseGithubIssueURL(ctx.Value(contextCfg.CtxKeyCreatedGithubIssueURL).(string))
	// Get current branch from the repo
	devBranchName, err := findBranchNameWithPrefix(ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string), issueNum)
	if err != nil {
		return err
	}
	// Build full repo URL
	repoURL := fmt.Sprintf("https://github.com/%s/%s", repoOwner, repoName)
	// Run gh issue develop --list command
	cmd := exec.Command("gh", "issue", "develop", "--list", "--repo", repoURL, issueNum)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check linked branches: %w", err)
	}

	// Check if current branch exists in the output
	if !strings.Contains(string(output), devBranchName) {
		return fmt.Errorf("current branch %s is not linked to issue #%s", devBranchName, issueNum)
	}

	return nil
}

type expectedCurrentBranchHasPrefix struct{}

func (e expectedCurrentBranchHasPrefix) Check(ctx context.Context) error {
	// Open the repository
	repo, err := git.PlainOpen(ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string))
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
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

// expectedPRBranchState checks:
// 1. If PR branch exists with correct naming pattern (-dev → -pr)
// 2. If PR branch contains notes with branch_type=2
// 3. If the PR branch has exactly one squashed commit
// 4. If the dev branch no longer exists (locally and remotely)
// 5. If an actual pull request was created in the upstream repo
type expectedPRBranchState struct{}

func (e expectedPRBranchState) Check(ctx context.Context) error {
	// 1. Check if PR branch exists with correct naming
	cloneRepoPath := ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string)
	if cloneRepoPath == "" {
		return fmt.Errorf("clone repo path not found in context")
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
	repo, err := git.PlainOpen(cloneRepoPath)
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
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
	cmd := exec.Command("git", "-C", cloneRepoPath, "checkout", expectedPRBranch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout PR branch: %w", err)
	}

	// Get notes from the branch
	stdout, stderr, err := new(goUtilsExec.PipedExec).
		Command("git", "-C", cloneRepoPath, "notes", "show").
		RunToStrings()

	if err != nil {
		return fmt.Errorf("failed to get git notes: %w, stderr: %s", err, stderr)
	}

	// Check if notes contain branch_type: 2
	if !strings.Contains(stdout, `"branch_type": 2`) && !strings.Contains(stdout, `"branch_type":2`) {
		return fmt.Errorf("PR branch notes do not contain branch_type: 2. Notes: %s", stdout)
	}

	// 3. Check if the PR branch has exactly one squashed commit
	// First get number of commits
	cmd = exec.Command("git", "-C", cloneRepoPath, "rev-list", "--count", "HEAD")
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
		cmd = exec.Command("git", "-C", cloneRepoPath, "log", "-1", "--pretty=%B")
		output, err = cmd.Output()
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

	// Check remotely
	stdout, stderr, err = new(goUtilsExec.PipedExec).
		Command("git", "-C", cloneRepoPath, "ls-remote", "--heads", "origin", devBranchName).
		RunToStrings()

	if err != nil {
		return fmt.Errorf("failed to check remote branches: %w, stderr: %s", err, stderr)
	}

	if stdout != "" {
		return fmt.Errorf("dev branch %s still exists on remote after PR creation", devBranchName)
	}

	// 5. Check if a real pull request was created in upstream repo
	// Extract repo owner and name from upstream remote
	upstream, err := repo.Remote("upstream")
	if err != nil {
		return fmt.Errorf("upstream remote not found: %w", err)
	}

	upstreamURL := upstream.Config().URLs[0]
	var owner, repoName string

	owner, repoName, err = gitCmds.ParseGitRemoteURL(upstreamURL)
	if err != nil {
		return fmt.Errorf("failed to parse upstream URL: %w", err)
	}
	// Use gh CLI to check if PR exists
	stdout, stderr, err = new(goUtilsExec.PipedExec).
		Command("gh", "pr", "list", "--repo", fmt.Sprintf("%s/%s", owner, repoName), "--head", expectedPRBranch, "--json", "number").
		RunToStrings()

	if err != nil {
		return fmt.Errorf("failed to list PRs: %w, stderr: %s", err, stderr)
	}

	// Parse JSON response
	var prList []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &prList); err != nil {
		return fmt.Errorf("failed to parse PR list: %w", err)
	}

	if len(prList) == 0 {
		return fmt.Errorf("no pull request found for branch %s", expectedPRBranch)
	}

	return nil
}

type expectedCommitsFromAnotherClone struct{}

func (e expectedCommitsFromAnotherClone) Check(ctx context.Context) error {
	// Check if the commit from another clone exists
	// Get the current branch
	repo, err := git.PlainOpen(ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string))
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
	}

	// Get the current branch
	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get the commit history
	commits, err := repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return fmt.Errorf("failed to get commit history: %w", err)
	}

	// Check if the commit from another clone exists
	commitFound := false
	err = commits.ForEach(func(c *object.Commit) error {
		if strings.Contains(c.Message, headerOfFilesInAnotherClone) {
			commitFound = true
			return nil
		}
		return nil
	})

	if !commitFound {
		return fmt.Errorf("commit from another clone not found")
	}

	return nil
}

type expectedRemoteBranch struct{}

func (e expectedRemoteBranch) Check(ctx context.Context) error {
	// Check if the remote branch exists
	cloneRepoPath := ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string)
	if cloneRepoPath == "" {
		return fmt.Errorf("clone repo path not found in context")
	}

	remoteBranchName, ok := ctx.Value(contextCfg.CtxKeyDevBranchName).(string)
	if !ok {
		return fmt.Errorf("remote branch name not found in context")
	}

	// Check if branch exists on the remote
	stdout, stderr, err := new(goUtilsExec.PipedExec).
		Command("git", "ls-remote", "--heads", "origin", remoteBranchName).
		WorkingDir(cloneRepoPath).
		RunToStrings()

	if err != nil {
		return fmt.Errorf("failed to check remote branches: %w: %s", err, stderr)
	}

	if stdout == "" {
		return fmt.Errorf("remote branch %s not found", remoteBranchName)
	}

	return nil
}
