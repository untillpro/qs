package systrun

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	// Setup upstream repo if needed
	if st.cfg.UpstreamState != RemoteStateNull {
		upstreamRepoURL := fmt.Sprintf("https://github.com/%s/%s.git",
			st.cfg.GHConfig.UpstreamAccount, st.repoName)

		if err := st.createUpstreamRepo(st.repoName, upstreamRepoURL); err != nil {
			return err
		}
	}

	// Setup fork repo if needed
	if st.cfg.ForkState != RemoteStateNull {
		forkRepoURL := fmt.Sprintf("https://github.com/%s/%s.git",
			st.cfg.GHConfig.ForkAccount, st.repoName)

		if err := st.createForkRepo(st.repoName, forkRepoURL); err != nil {
			return err
		}
	}

	// Determine which repo to clone based on test configuration
	var cloneURL string
	var authToken string

	if st.cfg.ForkState != RemoteStateNull {
		// Clone from fork if it exists
		cloneURL = fmt.Sprintf("https://github.com/%s/%s.git",
			st.cfg.GHConfig.ForkAccount, st.repoName)
		authToken = st.cfg.GHConfig.ForkToken
	} else if st.cfg.UpstreamState != RemoteStateNull {
		// Otherwise clone from upstream
		cloneURL = fmt.Sprintf("https://github.com/%s/%s.git",
			st.cfg.GHConfig.UpstreamAccount, st.repoName)
		authToken = st.cfg.GHConfig.UpstreamToken
	} else {
		return fmt.Errorf("cannot create test environment: both upstream and fork repos are null")
	}

	if err := st.cloneRepo(cloneURL, st.cloneRepoPath, authToken); err != nil {
		return err
	}

	// Configure remotes based on test scenario
	if err := st.configureRemotes(st.repoName); err != nil {
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

func ghAuthLogin(token string) error {
	// Connect stdin to pass the token to the gh process
	cmd := exec.Command("gh", "auth", "login", "--with-token")

	// Important: Connect stdout and stderr too
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	// Write token immediately
	_, err = stdin.Write([]byte(token))
	if err != nil {
		return fmt.Errorf("failed to write token to stdin: %w", err)
	}

	// Start the command first!
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start gh auth login: %w", err)
	}

	return stdin.Close() // IMPORTANT: Close stdin immediately after writing
}

// createUpstreamRepo creates the upstream repository
func (st *SystemTest) createUpstreamRepo(repoName, repoURL string) error {
	// GitHub Authentication
	if err := ghAuthLogin(st.cfg.GHConfig.UpstreamToken); err != nil {
		return fmt.Errorf("failed to authenticate with GitHub: %w", err)
	}
	//authCmd := exec.Command("gh", "auth", "login", "--with-token")
	//authCmd.Stdin = strings.NewReader(st.cfg.GHConfig.UpstreamToken)
	//if _, err := authCmd.CombinedOutput(); err != nil {
	//	return fmt.Errorf("failed to authenticate with GitHub: %v", err)
	//}

	// Use GitHub API to create repo
	cmd := exec.Command("gh", "repo", "create",
		fmt.Sprintf("%s/%s", st.cfg.GHConfig.UpstreamAccount, repoName),
		"--private")

	//cmd.Env = append(os.Environ(),
	//	fmt.Sprintf("GITHUB_TOKEN=%s", st.cfg.GHConfig.UpstreamToken))
	//
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
			"--private")

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

// checkExpectations checks expectations after command execution
func (st *SystemTest) checkExpectations() error {
	for _, expectation := range st.cfg.Expectations {
		if err := expectation.Check(st.cloneRepoPath); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
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

	// Check expectations
	if err := st.checkExpectations(); err != nil {
		return err
	}

	return nil
}
