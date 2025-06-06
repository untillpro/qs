package systrun

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// SystemTest represents a single system test for the qs utility
type SystemTest struct {
	t             *testing.T
	cfg           *TestConfig
	cloneRepoPath string
	repoName      string
}

// TestConfig contains all configuration for a system test
type TestConfig struct {
	TestID                    string
	GHConfig                  GithubConfig
	CommandConfig             CommandConfig
	UpstreamState             RemoteState
	ForkState                 RemoteState
	SyncState                 SyncState
	DevBranchExists           bool
	CreateGHIssueForDevBranch bool
	CheckoutOnBranch          string
	ExpectedStderr            string
	ExpectedStdout            string
	Expectations              []IExpectation
}

type CommandConfig struct {
	Command string
	Args    []string
	Stdin   string
}

// GithubConfig holds GitHub account and token information
type GithubConfig struct {
	UpstreamAccount string
	UpstreamToken   string
	ForkAccount     string
	ForkToken       string
}

// ExpectedDevBranch represents the expected state of the branch
type ExpectedDevBranch struct {
	BranchName string
	Exists     bool
}

func (e ExpectedDevBranch) Check(cloneRepoPath string) error {
	// Open the repository
	repo, err := git.PlainOpen(cloneRepoPath)
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
	}

	// Check if dev branch exists
	devRef := plumbing.NewBranchReferenceName(e.BranchName)
	_, err = repo.Reference(devRef, true)
	if err != nil {
		return fmt.Errorf("%s branch not found after dev command: %w", e.BranchName, err)
	}

	// Check if the local branch is tracking the remote branch
	cfg, err := repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get repo config: %w", err)
	}

	if branch, ok := cfg.Branches[e.BranchName]; ok {
		if branch.Remote != "origin" {
			return fmt.Errorf("%s branch is not tracking origin remote: %s", e.BranchName, branch.Remote)
		}
	} else {
		return fmt.Errorf("%s branch configuration not found", e.BranchName)
	}

	return nil
}

// ExpectedRemoteState represents the expected state of a remote
type ExpectedRemoteState struct {
	UpstreamRemoteState RemoteState
	ForkRemoteState     RemoteState
}

func (e ExpectedRemoteState) Check(cloneRepoPath string) error {
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

func (e ExpectedPullRequest) Check(cloneRepoPath string) error {
	// Open the repository
	repo, err := git.PlainOpen(cloneRepoPath)
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

func (e ExpectedDownloadResult) Check(cloneRepoPath string) error {
	// Compare local and remote branches to ensure they're in sync
	repo, err := git.PlainOpen(cloneRepoPath)
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

func (e ExpectedUploadResult) Check(cloneRepoPath string) error {
	// Similar to download but checks if the changes were pushed to remote
	repo, err := git.PlainOpen(cloneRepoPath)
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

func (e ExpectedForkExists) Check(cloneRepoPath string) error {
	// Implement the logic to check if the fork exists
	// get remotes of the local repo and check if remote, called origin, exists
	repo, err := git.PlainOpen(cloneRepoPath)
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
	}

	// Check if the remote named "origin" exists
	if _, err := repo.Remote("origin"); err != nil {
		return fmt.Errorf("origin remote not found after fork command: %w", err)
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
	IssueID string
}

// Check verifies if the branch is linked to a GitHub issue
func (e ExpectedBranchLinkedToIssue) Check(cloneRepoPath string) error {
	// Check if the branch linked to issue via `gh issue develop --list` command contains the issue number

	// Get current branch from the repo
	repo, err := git.PlainOpen(cloneRepoPath)
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
	}

	branches, err := repo.Branches()
	if err != nil {
		return fmt.Errorf("failed to get branches: %w", err)
	}

	// Find development branch name that starts with the issue ID
	devBranchName := ""
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		branchName := ref.Name().Short()
		if strings.HasPrefix(branchName, e.IssueID) {
			devBranchName = branchName
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to iterate through the branches: %w", err)
	}

	if devBranchName == "" {
		return fmt.Errorf("no branch found for issue ID %s", e.IssueID)
	}

	// Find upstream remote URL or fork remote to build GitHub issue URL
	upstreamRemote, err := repo.Remote("upstream")
	var repoGitURL string
	if err != nil {
		if !errors.Is(err, git.ErrRemoteNotFound) {
			return fmt.Errorf("failed to get upstream remote: %w", err)
		}

		originRemote, err := repo.Remote("origin")
		if err != nil && !errors.Is(err, git.ErrRemoteNotFound) {
			return fmt.Errorf("failed to get fork remote: %w", err)
		}

		repoGitURL = originRemote.Config().URLs[0]
	}

	if repoGitURL == "" {
		if upstreamRemote == nil {
			return fmt.Errorf("no upstream or fork remote found")
		}
		repoGitURL = upstreamRemote.Config().URLs[0]
	}

	// Build GitHub issue URL
	githubIssueURL := strings.TrimSuffix(repoGitURL, ".git") + "/issues/" + e.IssueID
	// extract repo and issue number from e.GithubIssueURL using regex
	re := regexp.MustCompile(`https://github\.com/([^/]+)/([^/]+)/issues/(\d+)`)
	matches := re.FindStringSubmatch(githubIssueURL)
	if matches == nil {
		return fmt.Errorf("invalid GitHub issue URL format: %s", githubIssueURL)
	}

	repoOwner := matches[1]
	repoName := matches[2]
	issueNum := matches[3]
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

type ExpectedDevBranchNameStartsWith struct {
	Prefix string
}

func (e ExpectedDevBranchNameStartsWith) Check(cloneRepoPath string) error {
	// Open the repository
	repo, err := git.PlainOpen(cloneRepoPath)
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
	}

	// Get the current branch
	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Check if the branch name starts with the expected prefix
	if !strings.HasPrefix(head.Name().Short(), e.Prefix) {
		return fmt.Errorf("branch name '%s' does not start with expected prefix '%s'", head.Name().Short(), e.Prefix)
	}

	return nil
}
