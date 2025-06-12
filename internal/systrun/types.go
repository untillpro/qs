package systrun

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	goUtilsExec "github.com/untillpro/goutils/exec"
	gitCmds "github.com/untillpro/qs/gitcmds"
)

// SystemTest represents a single system test for the qs utility
type SystemTest struct {
	cfg           *TestConfig
	cloneRepoPath string
	repoName      string
	qsExecRootCmd func(args []string) error
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
	Command string
	Args    []string
	Stdin   string
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

// ExpectedCurrentBranch represents the expected state of the branch
type ExpectedCurrentBranch struct {
}

func (e ExpectedCurrentBranch) Check(re *RuntimeEnvironment) error {
	currentBranch := gitCmds.GetCurrentBranchName(re.cloneRepoPath)
	if currentBranch != re.customBranchName {
		return fmt.Errorf("current branch '%s' does not match expected branch '%s'", currentBranch, re.customBranchName)
	}

	return nil
}

// ExpectedRemoteState represents the expected state of a remote
type ExpectedRemoteState struct {
	UpstreamRemoteState RemoteState
	ForkRemoteState     RemoteState
}

func (e ExpectedRemoteState) Check(_ *RuntimeEnvironment) error {
	// Implement the logic to check the remote state

	return nil
}

// ExpectedPullRequest represents the expected state of a pull request
type ExpectedPullRequest struct {
	Exists          bool
	Title           string
	ForkAccount     string
	UpstreamAccount string
}

func (e ExpectedPullRequest) Check(re *RuntimeEnvironment) error {
	// Open the repository
	repo, err := git.PlainOpen(re.cloneRepoPath)
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
type ExpectedDownloadResult struct {
}

func (e ExpectedDownloadResult) Check(re *RuntimeEnvironment) error {
	// Compare local and remote branches to ensure they're in sync
	repo, err := git.PlainOpen(re.cloneRepoPath)
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
type ExpectedUploadResult struct {
}

func (e ExpectedUploadResult) Check(re *RuntimeEnvironment) error {
	// Similar to download but checks if the changes were pushed to remote
	repo, err := git.PlainOpen(re.cloneRepoPath)
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
type ExpectedForkExists struct {
}

func (e ExpectedForkExists) Check(re *RuntimeEnvironment) error {
	// Implement the logic to check if the fork exists
	// get remotes of the local repo and check if remote, called origin, exists
	repo, err := git.PlainOpen(re.cloneRepoPath)
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

// ExpectedBranchLinkedToIssue represents the expected state of a branch linked to a GitHub issue
type ExpectedBranchLinkedToIssue struct {
}

// Check verifies if the branch is linked to a GitHub issue
func (e ExpectedBranchLinkedToIssue) Check(re *RuntimeEnvironment) error {
	// Get current branch from the repo
	repo, err := git.PlainOpen(re.cloneRepoPath)
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
	}

	// extract repo and issue number from e.createdGithubIssueURL using regex
	regExp := regexp.MustCompile(`https://github\.com/([^/]+)/([^/]+)/issues/(\d+)`)
	matches := regExp.FindStringSubmatch(re.createdGithubIssueURL)
	if matches == nil {
		return fmt.Errorf("invalid GitHub issue URL format: %s", re.createdGithubIssueURL)
	}
	// Extract repo owner, repo name, and issue number from the matches
	repoOwner := matches[1]
	repoName := matches[2]
	issueNum := matches[3]

	branches, err := repo.Branches()
	if err != nil {
		return fmt.Errorf("failed to get branches: %w", err)
	}

	// Find development branch name that starts with the issue ID
	devBranchName := ""
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		branchName := ref.Name().Short()
		if strings.HasPrefix(branchName, issueNum) {
			devBranchName = branchName
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to iterate through the branches: %w", err)
	}

	if devBranchName == "" {
		return fmt.Errorf("no branch found for issue ID %s", issueNum)
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

type ExpectedCurrentBranchHasPrefix struct {
}

func (e ExpectedCurrentBranchHasPrefix) Check(re *RuntimeEnvironment) error {
	// Open the repository
	repo, err := git.PlainOpen(re.cloneRepoPath)
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
	}

	// Get the current branch
	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Check if the branch name starts with the expected prefix
	if !strings.HasPrefix(head.Name().Short(), re.branchPrefix) {
		return fmt.Errorf("branch name '%s' does not start with expected prefix '%s'", head.Name().Short(), re.branchPrefix)
	}

	return nil
}

// ExpectedPRBranchState checks:
// 1. If PR branch exists with correct naming pattern (-dev → -pr)
// 2. If PR branch contains notes with branch_type=2
// 3. If the PR branch has exactly one squashed commit
// 4. If the dev branch no longer exists (locally and remotely)
// 5. If an actual pull request was created in the upstream repo
type ExpectedPRBranchState struct {
	DevBranchName string // Original dev branch name
}

func (e ExpectedPRBranchState) Check(re *RuntimeEnvironment) error {
	// 1. Check if PR branch exists with correct naming
	expectedPRBranch := ""
	if strings.HasSuffix(e.DevBranchName, "-dev") {
		expectedPRBranch = strings.TrimSuffix(e.DevBranchName, "-dev") + "-pr"
	} else {
		return fmt.Errorf("dev branch %s does not have expected -dev suffix", e.DevBranchName)
	}

	// Open the repository
	repo, err := git.PlainOpen(re.cloneRepoPath)
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
	cmd := exec.Command("git", "-C", re.cloneRepoPath, "checkout", expectedPRBranch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout PR branch: %w", err)
	}

	// Get notes from the branch
	stdout, stderr, err := new(goUtilsExec.PipedExec).
		Command("git", "-C", re.cloneRepoPath, "notes", "show").
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
	cmd = exec.Command("git", "-C", re.cloneRepoPath, "rev-list", "--count", "HEAD")
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
		cmd = exec.Command("git", "-C", re.cloneRepoPath, "log", "-1", "--pretty=%B")
		output, err = cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get last commit message: %w", err)
		}

		if !strings.Contains(string(output), "commit for notes") {
			return fmt.Errorf("last commit is not the expected empty commit for notes")
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
		if ref.Name().Short() == e.DevBranchName {
			devBranchExists = true
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error iterating branches: %w", err)
	}

	if devBranchExists {
		return fmt.Errorf("dev branch %s still exists locally after PR creation", e.DevBranchName)
	}

	// Check remotely
	stdout, stderr, err = new(goUtilsExec.PipedExec).
		Command("git", "-C", re.cloneRepoPath, "ls-remote", "--heads", "origin", e.DevBranchName).
		RunToStrings()

	if err != nil {
		return fmt.Errorf("failed to check remote branches: %w, stderr: %s", err, stderr)
	}

	if stdout != "" {
		return fmt.Errorf("dev branch %s still exists on remote after PR creation", e.DevBranchName)
	}

	// 5. Check if a real pull request was created in upstream repo
	// Extract repo owner and name from upstream remote
	upstream, err := repo.Remote("upstream")
	if err != nil {
		return fmt.Errorf("upstream remote not found: %w", err)
	}

	upstreamURL := upstream.Config().URLs[0]
	var owner, repoName string

	// Handle both HTTPS and SSH URLs
	if strings.HasPrefix(upstreamURL, "https://") {
		urlParts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(upstreamURL, "https://github.com/"), ".git"), "/")
		if len(urlParts) < 2 {
			return fmt.Errorf("invalid upstream URL format: %s", upstreamURL)
		}
		owner = urlParts[0]
		repoName = urlParts[1]
	} else if strings.Contains(upstreamURL, "git@github.com:") {
		urlParts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(upstreamURL, "git@github.com:"), ".git"), "/")
		if len(urlParts) < 2 {
			return fmt.Errorf("invalid upstream SSH URL format: %s", upstreamURL)
		}
		owner = urlParts[0]
		repoName = urlParts[1]
	} else {
		return fmt.Errorf("unsupported upstream URL format: %s", upstreamURL)
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
