package systrun

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// checkPrerequisites ensures all required tools are available
func (st *SystemTest) checkPrerequisites() error {
	// Check for git
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found in PATH: %w", err)
	}

	// Check for gh
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh not found in PATH: %w", err)
	}

	// Check for qs
	if _, err := exec.LookPath("qs"); err != nil {
		return fmt.Errorf("qs not found in PATH: %w", err)
	}

	// Check command validity
	if err := st.checkCommand(); err != nil {
		return err
	}

	return nil
}

// checkCommand validates the command to be executed
func (st *SystemTest) checkCommand() error {
	switch st.cfg.Command {
	case "fork", "dev", "pr", "d", "u":
		return nil
	default:
		return fmt.Errorf("unknown command: %s", st.cfg.Command)
	}
}

// createTestEnvironment sets up the test repositories based on configuration
func (st *SystemTest) createTestEnvironment() error {
	// Generate unique repo name
	timestamp := time.Now().Format("060102150405") // YYMMDDhhmmss
	repoName := fmt.Sprintf("%s-%s", st.cfg.TestID, timestamp)

	// Setup upstream repo if needed
	if st.cfg.UpstreamState != RemoteStateNull {
		upstreamRepoURL := fmt.Sprintf("https://github.com/%s/%s.git",
			st.cfg.GHConfig.UpstreamAccount, repoName)

		if err := st.createUpstreamRepo(repoName, upstreamRepoURL); err != nil {
			return err
		}
	}

	// Setup fork repo if needed
	if st.cfg.ForkState != RemoteStateNull {
		forkRepoURL := fmt.Sprintf("https://github.com/%s/%s.git",
			st.cfg.GHConfig.ForkAccount, repoName)

		if err := st.createForkRepo(repoName, forkRepoURL); err != nil {
			return err
		}
	}

	// Clone the appropriate repo
	clonePath := filepath.Join(".testdata", repoName)
	st.cloneRepoPath = clonePath

	// Determine which repo to clone based on test configuration
	var cloneURL string
	var authToken string

	if st.cfg.ForkState != RemoteStateNull {
		// Clone from fork if it exists
		cloneURL = fmt.Sprintf("https://github.com/%s/%s.git",
			st.cfg.GHConfig.ForkAccount, repoName)
		authToken = st.cfg.GHConfig.ForkToken
	} else if st.cfg.UpstreamState != RemoteStateNull {
		// Otherwise clone from upstream
		cloneURL = fmt.Sprintf("https://github.com/%s/%s.git",
			st.cfg.GHConfig.UpstreamAccount, repoName)
		authToken = st.cfg.GHConfig.UpstreamToken
	} else {
		return fmt.Errorf("cannot create test environment: both upstream and fork repos are null")
	}

	if err := st.cloneRepo(cloneURL, clonePath, authToken); err != nil {
		return err
	}

	// Configure remotes based on test scenario
	if err := st.configureRemotes(repoName); err != nil {
		return err
	}

	// Setup dev branch if needed
	if st.cfg.DevBranchExists {
		if err := st.setupDevBranch(); err != nil {
			return err
		}
	}

	return nil
}

// createUpstreamRepo creates the upstream repository
func (st *SystemTest) createUpstreamRepo(repoName, repoURL string) error {
	// Use GitHub API to create repo
	cmd := exec.Command("gh", "repo", "create",
		fmt.Sprintf("%s/%s", st.cfg.GHConfig.UpstreamAccount, repoName),
		"--public")

	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GITHUB_TOKEN=%s", st.cfg.GHConfig.UpstreamToken))

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create upstream repo: %w\nOutput: %s", err, output)
	}

	// Initialize repo with a README if state should be OK
	if st.cfg.UpstreamState == RemoteStateOK {
		// Create temp dir for initial commit
		tempDir, err := os.MkdirTemp("", "repo-init-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		defer os.RemoveAll(tempDir)

		// Initialize git repo
		repo, err := git.PlainInit(tempDir, false)
		if err != nil {
			return fmt.Errorf("failed to initialize git repo: %w", err)
		}

		// Create README file
		readmePath := filepath.Join(tempDir, "README.md")
		readmeContent := "# Test Repository\n\nThis is a test repository created for system tests."
		if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
			return fmt.Errorf("failed to create README file: %w", err)
		}

		// Get worktree
		wt, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}

		// Add README to index
		if _, err := wt.Add("README.md"); err != nil {
			return fmt.Errorf("failed to add README to index: %w", err)
		}

		// Commit changes
		_, err = wt.Commit("Initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "System Test",
				Email: "test@example.com",
				When:  time.Now(),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to commit changes: %w", err)
		}

		// Set remote
		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{repoURL},
		})
		if err != nil {
			return fmt.Errorf("failed to set remote: %w", err)
		}

		// Push changes
		err = repo.Push(&git.PushOptions{
			RemoteName: "origin",
			Auth: &http.BasicAuth{
				Username: "x-access-token",
				Password: st.cfg.GHConfig.UpstreamToken,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to push changes: %w", err)
		}
	}

	return nil
}

// createForkRepo creates or configures the fork repository
func (st *SystemTest) createForkRepo(repoName, repoURL string) error {
	if st.cfg.UpstreamState != RemoteStateNull {
		// Fork the upstream repo
		cmd := exec.Command("gh", "repo", "fork",
			fmt.Sprintf("%s/%s", st.cfg.GHConfig.UpstreamAccount, repoName),
			"--clone=false")

		cmd.Env = append(os.Environ(),
			fmt.Sprintf("GITHUB_TOKEN=%s", st.cfg.GHConfig.ForkToken))

		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to fork upstream repo: %w\nOutput: %s", err, output)
		}
	} else {
		// Create an independent repo
		cmd := exec.Command("gh", "repo", "create",
			fmt.Sprintf("%s/%s", st.cfg.GHConfig.ForkAccount, repoName),
			"--public")

		cmd.Env = append(os.Environ(),
			fmt.Sprintf("GITHUB_TOKEN=%s", st.cfg.GHConfig.ForkToken))

		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create fork repo: %w\nOutput: %s", err, output)
		}

		// Initialize repo with a README if state should be OK
		if st.cfg.ForkState == RemoteStateOK {
			// Create temp dir for initial commit
			tempDir, err := os.MkdirTemp("", "repo-init-*")
			if err != nil {
				return fmt.Errorf("failed to create temp directory: %w", err)
			}
			defer os.RemoveAll(tempDir)

			// Initialize git repo
			repo, err := git.PlainInit(tempDir, false)
			if err != nil {
				return fmt.Errorf("failed to initialize git repo: %w", err)
			}

			// Create README file
			readmePath := filepath.Join(tempDir, "README.md")
			readmeContent := "# Test Repository\n\nThis is a test repository created for system tests."
			if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
				return fmt.Errorf("failed to create README file: %w", err)
			}

			// Get worktree
			wt, err := repo.Worktree()
			if err != nil {
				return fmt.Errorf("failed to get worktree: %w", err)
			}

			// Add README to index
			if _, err := wt.Add("README.md"); err != nil {
				return fmt.Errorf("failed to add README to index: %w", err)
			}

			// Commit changes
			_, err = wt.Commit("Initial commit", &git.CommitOptions{
				Author: &object.Signature{
					Name:  "System Test",
					Email: "test@example.com",
					When:  time.Now(),
				},
			})
			if err != nil {
				return fmt.Errorf("failed to commit changes: %w", err)
			}

			// Set remote
			_, err = repo.CreateRemote(&config.RemoteConfig{
				Name: "origin",
				URLs: []string{repoURL},
			})
			if err != nil {
				return fmt.Errorf("failed to set remote: %w", err)
			}

			// Push changes
			err = repo.Push(&git.PushOptions{
				RemoteName: "origin",
				Auth: &http.BasicAuth{
					Username: "x-access-token",
					Password: st.cfg.GHConfig.ForkToken,
				},
			})
			if err != nil {
				return fmt.Errorf("failed to push changes: %w", err)
			}
		}
	}

	return nil
}

// cloneRepo clones a repository to the local machine
func (st *SystemTest) cloneRepo(repoURL, clonePath, token string) error {
	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(clonePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Clone the repository
	cloneOpts := &git.CloneOptions{
		URL: repoURL,
		Auth: &http.BasicAuth{
			Username: "x-access-token",
			Password: token,
		},
	}

	_, err := git.PlainClone(clonePath, false, cloneOpts)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// configureRemotes sets up the remote configuration in the cloned repository
func (st *SystemTest) configureRemotes(repoName string) error {
	// Open the repository
	repo, err := git.PlainOpen(st.cloneRepoPath)
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
	}

	// Configure remotes based on test scenario
	switch {
	case st.cfg.ForkState != RemoteStateNull && st.cfg.UpstreamState != RemoteStateNull:
		// Both upstream and fork exist, configure both remotes
		_, err := repo.Remote("origin")
		if err != nil {
			// Add origin remote pointing to fork
			forkURL := fmt.Sprintf("https://github.com/%s/%s.git",
				st.cfg.GHConfig.ForkAccount, repoName)
			_, err = repo.CreateRemote(&config.RemoteConfig{
				Name: "origin",
				URLs: []string{forkURL},
			})
			if err != nil {
				return fmt.Errorf("failed to create origin remote: %w", err)
			}
		}

		_, err = repo.Remote("upstream")
		if err != nil {
			// Add upstream remote
			upstreamURL := fmt.Sprintf("https://github.com/%s/%s.git",
				st.cfg.GHConfig.UpstreamAccount, repoName)
			_, err = repo.CreateRemote(&config.RemoteConfig{
				Name: "upstream",
				URLs: []string{upstreamURL},
			})
			if err != nil {
				return fmt.Errorf("failed to create upstream remote: %w", err)
			}
		}

	case st.cfg.ForkState == RemoteStateNull && st.cfg.UpstreamState != RemoteStateNull:
		// Only upstream exists, make sure origin points to upstream
		_, err := repo.Remote("origin")
		if err != nil {
			upstreamURL := fmt.Sprintf("https://github.com/%s/%s.git",
				st.cfg.GHConfig.UpstreamAccount, repoName)
			_, err = repo.CreateRemote(&config.RemoteConfig{
				Name: "origin",
				URLs: []string{upstreamURL},
			})
			if err != nil {
				return fmt.Errorf("failed to create origin remote: %w", err)
			}
		}
	}

	return nil
}

// setupDevBranch creates and configures the dev branch
func (st *SystemTest) setupDevBranch() error {
	// Open the repository
	repo, err := git.PlainOpen(st.cloneRepoPath)
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
	}

	// Get worktree
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Create dev branch from HEAD
	headRef, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Create local dev branch
	branchRef := plumbing.NewBranchReferenceName("dev")
	ref := plumbing.NewHashReference(branchRef, headRef.Hash())
	if err := repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("failed to create dev branch: %w", err)
	}

	// Checkout the dev branch
	err = wt.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout dev branch: %w", err)
	}

	// Push dev branch to remote (if fork exists)
	if st.cfg.ForkState != RemoteStateNull {
		err = repo.Push(&git.PushOptions{
			RemoteName: "origin",
			Auth: &http.BasicAuth{
				Username: "x-access-token",
				Password: st.cfg.GHConfig.ForkToken,
			},
			RefSpecs: []config.RefSpec{
				config.RefSpec(fmt.Sprintf("refs/heads/dev:refs/heads/dev")),
			},
		})
		if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return fmt.Errorf("failed to push dev branch to fork: %w", err)
		}
	}

	return nil
}

// runCommand executes the specified qs command
func (st *SystemTest) runCommand() (string, string, error) {
	// Change to the clone repo directory
	originalDir, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("failed to get current directory: %w", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(st.cloneRepoPath); err != nil {
		return "", "", fmt.Errorf("failed to change directory to clone repo: %w", err)
	}

	// Prepare command
	cmd := exec.Command(st.cfg.Command, st.cfg.Args...)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()

	return stdout.String(), stderr.String(), err
}

// validateOutput checks the command output against expected values
func (st *SystemTest) validateOutput(stdout, stderr string) error {
	// Check stdout if specified
	if st.cfg.ExpectedStdout != "" {
		if !strings.Contains(stdout, st.cfg.ExpectedStdout) {
			return fmt.Errorf("expected stdout to contain: %q, got: %q",
				st.cfg.ExpectedStdout, stdout)
		}
	}

	// Check stderr if specified
	if st.cfg.ExpectedStderr != "" {
		if !strings.Contains(stderr, st.cfg.ExpectedStderr) {
			return fmt.Errorf("expected stderr to contain: %q, got: %q",
				st.cfg.ExpectedStderr, stderr)
		}
	}

	return nil
}

// validateCommandResult validates the state after command execution
func (st *SystemTest) validateCommandResult() error {
	switch st.cfg.Command {
	case "fork":
		return st.validateForkResult()
	case "dev":
		return st.validateDevResult()
	case "pr":
		return st.validatePRResult()
	case "d":
		return st.validateDownloadResult()
	case "u":
		return st.validateUploadResult()
	default:
		return fmt.Errorf("unknown command: %s", st.cfg.Command)
	}
}

// validateForkResult checks the repository state after a fork operation
func (st *SystemTest) validateForkResult() error {
	// Open the repository
	repo, err := git.PlainOpen(st.cloneRepoPath)
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
	}

	// Check remotes configuration
	origin, err := repo.Remote("origin")
	if err != nil {
		return fmt.Errorf("origin remote not found after fork command: %w", err)
	}

	// Verify origin points to fork
	expectedForkURL := fmt.Sprintf("https://github.com/%s/", st.cfg.GHConfig.ForkAccount)
	if !strings.Contains(origin.Config().URLs[0], expectedForkURL) {
		return fmt.Errorf("origin remote does not point to fork: %s", origin.Config().URLs[0])
	}

	// Verify upstream remote exists and points to upstream
	upstream, err := repo.Remote("upstream")
	if err != nil {
		return fmt.Errorf("upstream remote not found after fork command: %w", err)
	}

	expectedUpstreamURL := fmt.Sprintf("https://github.com/%s/", st.cfg.GHConfig.UpstreamAccount)
	if !strings.Contains(upstream.Config().URLs[0], expectedUpstreamURL) {
		return fmt.Errorf("upstream remote does not point to upstream: %s", upstream.Config().URLs[0])
	}

	return nil
}

// validateDevResult checks the repository state after a dev operation
func (st *SystemTest) validateDevResult() error {
	// Open the repository
	repo, err := git.PlainOpen(st.cloneRepoPath)
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
	}

	// Check if dev branch exists
	devRef := plumbing.NewBranchReferenceName("dev")
	_, err = repo.Reference(devRef, true)
	if err != nil {
		return fmt.Errorf("dev branch not found after dev command: %w", err)
	}

	// Check if the local branch is tracking the remote branch
	cfg, err := repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get repo config: %w", err)
	}

	if branch, ok := cfg.Branches["dev"]; ok {
		if branch.Remote != "origin" {
			return fmt.Errorf("dev branch is not tracking origin remote: %s", branch.Remote)
		}
	} else {
		return fmt.Errorf("dev branch configuration not found")
	}

	return nil
}

// validatePRResult checks if a pull request was created
func (st *SystemTest) validatePRResult() error {
	// Use GitHub API to check if PR was created
	cmd := exec.Command("gh", "pr", "list",
		"--repo", fmt.Sprintf("%s/%s", st.cfg.GHConfig.UpstreamAccount, st.getRepoName()),
		"--head", fmt.Sprintf("%s:dev", st.cfg.GHConfig.ForkAccount),
		"--json", "number")

	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GITHUB_TOKEN=%s", st.cfg.GHConfig.UpstreamToken))

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check pull requests: %w", err)
	}

	// Parse JSON output
	type prList struct {
		Number int `json:"number"`
	}

	var prs []prList
	if err := json.Unmarshal(output, &prs); err != nil {
		return fmt.Errorf("failed to parse PR list: %w", err)
	}

	if len(prs) == 0 {
		return fmt.Errorf("no pull request created")
	}

	return nil
}

// validateDownloadResult checks the repository state after a download operation
func (st *SystemTest) validateDownloadResult() error {
	// Compare local and remote branches to ensure they're in sync
	repo, err := git.PlainOpen(st.cloneRepoPath)
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

// validateUploadResult checks the repository state after an upload operation
func (st *SystemTest) validateUploadResult() error {
	// Similar to download but checks if the changes were pushed to remote
	repo, err := git.PlainOpen(st.cloneRepoPath)
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

// getRepoName extracts the repository name from the clone path
func (st *SystemTest) getRepoName() string {
	return filepath.Base(st.cloneRepoPath)
}

// Run executes the complete system test
func (st *SystemTest) Run() error {
	// Check prerequisites
	if err := st.checkPrerequisites(); err != nil {
		return err
	}

	// Create test environment
	if err := st.createTestEnvironment(); err != nil {
		return err
	}

	// Run the command
	stdout, stderr, err := st.runCommand()
	if err != nil {
		return err
	}

	// Validate output
	if err := st.validateOutput(stdout, stderr); err != nil {
		return err
	}

	// Validate command result
	if err := st.validateCommandResult(); err != nil {
		return err
	}

	return nil
}

// generateCloneRepoPath generates a unique path for the clone repo
func generateCloneRepoPath() string {
	timestamp := time.Now().Format("060102150405") // YYMMDDhhmmss
	repoName := fmt.Sprintf("qs-test-%s", timestamp)

	return filepath.Join(".testdata", repoName)
}
