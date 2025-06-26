package systrun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	notesPkg "github.com/untillpro/qs/internal/notes"
	"github.com/untillpro/qs/internal/types"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	goUtilsExec "github.com/untillpro/goutils/exec"
	"github.com/untillpro/qs/gitcmds"
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
	Expectations   []ExpectationFunc
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
	currentBranch := gitcmds.GetCurrentBranchName(ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string))
	customBranchName := ctx.Value(contextCfg.CtxKeyCustomBranchName).(string)
	if currentBranch != customBranchName {
		return fmt.Errorf("current branch '%s' does not match expected branch '%s'", currentBranch, customBranchName)
	}

	return nil
}

// ExpectationCloneIsSyncedWithFork checks if the clone is synchronized with the fork
func ExpectationCloneIsSyncedWithFork(ctx context.Context) error {
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

// ExpectationForkExists represents the expected state of a fork
func ExpectationForkExists(ctx context.Context) error {
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

// ExpectationBranchLinkedToIssue represents the expected state of a branch linked to a GitHub issue
func ExpectationBranchLinkedToIssue(ctx context.Context) error {
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

// ExpectationCurrentBranchHasPrefix checks if the current branch has the expected prefix
func ExpectationCurrentBranchHasPrefix(ctx context.Context) error {
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

	// Check remotely in origin
	stdout, stderr, err = new(goUtilsExec.PipedExec).
		Command("git", "-C", cloneRepoPath, "ls-remote", "--heads", "origin", devBranchName).
		RunToStrings()

	if err != nil {
		return fmt.Errorf("failed to check remote branches: %w, stderr: %s", err, stderr)
	}

	if stdout != "" {
		return fmt.Errorf("dev branch %s still exists on origin after PR creation", devBranchName)
	}

	// Check remotely in upstream
	stdout, stderr, err = new(goUtilsExec.PipedExec).
		Command("git", "-C", cloneRepoPath, "ls-remote", "--heads", "upstream", devBranchName).
		RunToStrings()

	if err != nil {
		return fmt.Errorf("failed to check remote branches: %w, stderr: %s", err, stderr)
	}

	if stdout != "" {
		return fmt.Errorf("dev branch %s still exists on upstream after PR creation", devBranchName)
	}

	// 5. Check if a real pull request was created in upstream repo
	// Extract repo owner and name from upstream remote
	upstreamRemote, err := repo.Remote("upstream")
	if err != nil {
		return fmt.Errorf("upstream remote not found: %w", err)
	}

	upstreamRemoteURL := upstreamRemote.Config().URLs[0]
	var owner, repoName string

	owner, repoName, _, err = gitcmds.ParseGitRemoteURL(upstreamRemoteURL)
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

// ExpectationRemoteBranchWithCommitMessage checks if the remote branch exists and has specific commit message
func ExpectationRemoteBranchWithCommitMessage(ctx context.Context) error {
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

	// check that remote branch has specific commit message
	stdout, stderr, err = new(goUtilsExec.PipedExec).
		Command("git", "log", "-1", "--pretty=%B", remoteBranchName).
		WorkingDir(cloneRepoPath).
		RunToStrings()

	if err != nil {
		return fmt.Errorf("failed to get last commit message: %w: %s", err, stderr)
	}

	commitMessage, ok := ctx.Value(contextCfg.CtxKeyCommitMessage).(string)
	if !ok {
		return fmt.Errorf("commit message not found in context")
	}

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
		return fmt.Errorf("clone repo path not found in context")
	}

	remoteBranchName, ok := ctx.Value(contextCfg.CtxKeyDevBranchName).(string)
	if !ok {
		return fmt.Errorf("remote branch name not found in context")
	}

	remoteURL, err := getRemoteUrlByName(cloneRepoPath, "origin")
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
	cloneCmd := exec.Command("git", "clone", remoteURL)
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
	notes, ok := gitcmds.GetNotes(tempClonePath)
	if !ok {
		return errors.New("Error: No notes found in dev branch")
	}

	if len(notes) == 0 {
		return fmt.Errorf("no notes downloaded")
	}

	// Step 7: Check if notes are of correct type
	notesObj, ok := notesPkg.Deserialize(notes)
	if !ok {
		return errors.New("error: No notes found in dev branch")
	}

	if notesObj.BranchType != types.BranchTypeDev {
		return fmt.Errorf("notes downloaded but branch type is not dev")
	}

	return nil
}

// ExpectationPrFromCloneIsSucceeded checks if PR from clone is successful
func ExpectationPrFromCloneIsSucceeded(ctx context.Context) error {
	// Step 1: Get the remote URL and repo name
	cloneRepoPath := ctx.Value(contextCfg.CtxKeyCloneRepoPath).(string)
	if cloneRepoPath == "" {
		return fmt.Errorf("clone repo path not found in context")
	}

	remoteBranchName, ok := ctx.Value(contextCfg.CtxKeyDevBranchName).(string)
	if !ok {
		return fmt.Errorf("remote branch name not found in context")
	}

	remoteOriginURL, err := getRemoteUrlByName(cloneRepoPath, "origin")
	if err != nil {
		return fmt.Errorf("failed to get oririn remote URL: %w", err)
	}

	remoteUpstreamURL, err := getRemoteUrlByName(cloneRepoPath, "upstream")
	if err != nil {
		return fmt.Errorf("failed to get upstream remote URL: %w", err)
	}

	forkAccount, repo, forkToken, err := gitcmds.ParseGitRemoteURL(remoteOriginURL)
	if err != nil {
		return err
	}

	upstreamAccount, repo, upstreamToken, err := gitcmds.ParseGitRemoteURL(remoteUpstreamURL)
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
	cloneCmd := exec.Command("git", "clone", remoteOriginURL)
	cloneCmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", forkToken))
	cloneCmd.Dir = tempPath

	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone repository: %w, output: %s", err, output)
	}

	// Step 3.1: Configure remotes in temp clone
	if err := gitcmds.CreateRemote(
		tempClonePath,
		"upstream",
		upstreamAccount,
		upstreamToken,
		repo,
		true,
	); err != nil {
		return err
	}

	if err := gitcmds.CreateRemote(
		tempClonePath,
		"origin",
		forkAccount,
		forkToken,
		repo,
		false,
	); err != nil {
		return err
	}

	// Step 4: Checkout on remote branch
	if err := checkoutOnBranch(tempClonePath, remoteBranchName); err != nil {
		return err
	}

	// Step 4.1: Commit some changes
	if err := commitFiles(tempClonePath, true, "", 4); err != nil {
		return err
	}

	// Step 5: Run `qs pr`
	if err := gitcmds.Pr(tempClonePath, false); err != nil {
		return err
	}

	// Step 6: Check if notes are downloaded
	notes, ok := gitcmds.GetNotes(tempClonePath)
	if !ok {
		return errors.New("Error: No notes found in dev branch")
	}

	if len(notes) == 0 {
		return fmt.Errorf("no notes downloaded")
	}

	// Step 7: Check if notes are of correct type
	notesObj, ok := notesPkg.Deserialize(notes)
	if !ok {
		return errors.New("error: No notes found in dev branch")
	}

	if notesObj.BranchType != types.BranchTypePr {
		return fmt.Errorf("notes downloaded but branch type is not pr")
	}

	return nil
}
