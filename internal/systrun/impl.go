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

	"github.com/atotto/clipboard"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/uuid"
	"github.com/untillpro/qs/internal/commands"
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
	switch st.cfg.CommandConfig.Command {
	case commands.CommandNameFork,
		commands.CommandNameDev,
		commands.CommandNamePR,
		commands.CommandNameD,
		commands.CommandNameU:
		return nil
	default:
		return fmt.Errorf("unknown command: %s", st.cfg.CommandConfig)
	}
}

// createTestEnvironment sets up the test repositories based on configuration
// Returns:
// - RuntimeEnvironment with fields populated for the test
func (st *SystemTest) createTestEnvironment() (*RuntimeEnvironment, error) {
	re := &RuntimeEnvironment{
		cloneRepoPath: st.cloneRepoPath,
	}

	// Setup upstream repo if needed
	if st.cfg.UpstreamState != RemoteStateNull {
		upstreamRepoURL := buildRemoteURL(st.cfg.GHConfig.UpstreamAccount, st.cfg.GHConfig.UpstreamToken, st.repoName)
		if err := st.createUpstreamRepo(st.repoName, upstreamRepoURL); err != nil {
			return nil, err
		}
	}

	// Setup fork repo if needed
	if st.cfg.ForkState != RemoteStateNull {
		forkRepoURL := buildRemoteURL(st.cfg.GHConfig.ForkAccount, st.cfg.GHConfig.ForkToken, st.repoName)
		if err := st.createForkRepo(st.repoName, forkRepoURL); err != nil {
			return nil, err
		}
	}

	// Determine which repo to clone based on test configuration
	var cloneURL string
	var authToken string

	switch {
	case st.cfg.ForkState != RemoteStateNull:
		// Clone from fork if it exists
		cloneURL = fmt.Sprintf(remoteGithubRepoURLTemplate, st.cfg.GHConfig.ForkAccount, st.repoName)
		authToken = st.cfg.GHConfig.ForkToken
	case st.cfg.UpstreamState != RemoteStateNull:
		// Otherwise clone from upstream
		cloneURL = fmt.Sprintf(remoteGithubRepoURLTemplate, st.cfg.GHConfig.UpstreamAccount, st.repoName)
		authToken = st.cfg.GHConfig.UpstreamToken
	default:
		return nil, fmt.Errorf("cannot create test environment: both upstream and fork repos are null")
	}

	// Need some time to ensure the repo is created
	// TODO: add check in loop with deadline instead of sleep
	time.Sleep(time.Millisecond * 2000)
	if err := st.cloneRepo(cloneURL, st.cloneRepoPath, authToken); err != nil {
		return nil, err
	}

	// Configure remotes based on test scenario
	if err := st.configureRemotes(st.repoName); err != nil {
		return nil, err
	}

	// Setup dev branch if needed
	if err := st.setupDevBranch(); err != nil {
		return nil, err
	}

	if err := st.processClipboardContent(re); err != nil {
		return nil, fmt.Errorf("failed to process clipboard content: %w", err)
	}

	return re, nil
}

func (st *SystemTest) processClipboardContent(re *RuntimeEnvironment) error {
	var (
		clipboardContent string
		err              error
	)

	switch st.cfg.ClipboardContent {
	case ClipboardContentEmpty:
		return nil
	case ClipboardContentCustom:
		clipboardContent = st.getCustomClipboardContent()
		re.branchName = clipboardContent
	case ClipboardContentUnavailableGithubIssue:
		clipboardContent = fmt.Sprintf("https://github.com/%s/%s/issues/abc",
			st.cfg.GHConfig.UpstreamAccount,
			uuid.New().String(),
		)
	case ClipboardContentGithubIssue:
		clipboardContent, err = st.createGitHubIssue()
		if err != nil {
			return err
		}
		re.createdGithubIssueURL = clipboardContent
	case ClipboardContentJiraTicket:
		clipboardContent = os.Getenv("JIRA_TICKET_URL")
		if clipboardContent == "" {
			return errors.New("JIRA_TICKET_URL environment variable not set, skipping test")
		}

		jiraTicketID, ok := commands.GetJiraTicketIDFromArgs(clipboardContent)
		if !ok {
			return fmt.Errorf("invalid JIRA ticket URL: %s", clipboardContent)
		}

		re.branchPrefix = jiraTicketID
	default:
		return fmt.Errorf("unknown clipboard content type: %s", st.cfg.ClipboardContent)
	}

	if err := clipboard.WriteAll(clipboardContent); err != nil {
		return fmt.Errorf("failed to write custom clipboard content: %w", err)
	}

	return nil
}

func (st *SystemTest) getCustomClipboardContent() string {
	return "my-cool-clipboard-content"
}

// createGitHubIssue creates a GitHub issue for the dev branch if configured
func (st *SystemTest) createGitHubIssue() (string, error) {
	// Create issue title based on test ID
	issueTitle := fmt.Sprintf("Test automation issue for %s", st.cfg.TestID)
	issueBody := fmt.Sprintf("Automated test issue created by QS system test framework")

	// Run gh issue create command
	cmd := exec.Command("gh", "issue", "create",
		"--title", issueTitle,
		"--body", issueBody,
		"--repo", fmt.Sprintf("https://github.com/%s/%s", st.cfg.GHConfig.UpstreamAccount, st.repoName))

	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GITHUB_TOKEN=%s", st.cfg.GHConfig.UpstreamToken))

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to create GitHub issue: %w", err)
	}

	issueURL := strings.TrimSpace(string(output))

	return issueURL, nil
}

func (st *SystemTest) checkoutOnBranch(branchName string) error {
	if branchName == "" {
		return nil
	}

	repo, err := git.PlainOpen(st.cloneRepoPath)
	if err != nil {
		fmt.Println("Failed to open repo:", err)
		os.Exit(1)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		fmt.Println("Failed to get worktree:", err)
		os.Exit(1)
	}

	return worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
		Create: false, // true if the branch doesn't exist locally
		Force:  false, // true to override uncommitted changes
	})
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
	cmd := exec.Command("gh", "repo", "create",
		fmt.Sprintf("%s/%s", st.cfg.GHConfig.UpstreamAccount, repoName),
		"--public")

	cmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", st.cfg.GHConfig.UpstreamToken))

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
		repo, err := git.PlainInitWithOptions(tempDir, &git.PlainInitOptions{
			Bare: false,
			InitOptions: git.InitOptions{
				DefaultBranch: plumbing.Main,
			},
		})
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
		cmd := exec.Command(
			"gh",
			"repo",
			"fork",
			fmt.Sprintf("%s/%s", st.cfg.GHConfig.UpstreamAccount, repoName),
			"--clone=false",
		)

		cmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", st.cfg.GHConfig.ForkToken))

		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to fork upstream repo: %w\nOutput: %s", err, output)
		}
	} else {
		// Create an independent repo
		cmd := exec.Command(
			"gh",
			"repo",
			"create",
			fmt.Sprintf("%s/%s", st.cfg.GHConfig.ForkAccount, repoName),
			"--public",
		)

		cmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", st.cfg.GHConfig.ForkToken))

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
			repo, err := git.PlainInitWithOptions(tempDir, &git.PlainInitOptions{
				Bare: false,
				InitOptions: git.InitOptions{
					DefaultBranch: plumbing.Main,
				},
			})
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

	if _, err := git.PlainClone(clonePath, false, cloneOpts); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

func buildRemoteURL(account, token, repoName string) string {
	return "https://" + account + ":" + token + "@github.com/" + account + "/" + repoName + ".git"
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
			forkURL := buildRemoteURL(st.cfg.GHConfig.ForkAccount, st.cfg.GHConfig.ForkToken, repoName)
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
			upstreamURL := buildRemoteURL(st.cfg.GHConfig.UpstreamAccount, st.cfg.GHConfig.UpstreamToken, repoName)
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
			upstreamURL := buildRemoteURL(st.cfg.GHConfig.UpstreamAccount, st.cfg.GHConfig.UpstreamToken, repoName)
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
	if st.cfg.DevBranchState == DevBranchStateNotExists {
		return nil // No dev branch needed
	}

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

	// Checkout the dev branch if it should be current
	if st.cfg.DevBranchState == DevBranchStateExistsAndCurrent {
		err = wt.Checkout(&git.CheckoutOptions{
			Branch: branchRef,
		})
		if err != nil {
			return fmt.Errorf("failed to checkout dev branch: %w", err)
		}
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
	qsArgs := make([]string, 0, len(st.cfg.CommandConfig.Args)+1)
	qsArgs = append(qsArgs, st.cfg.CommandConfig.Command)
	qsArgs = append(qsArgs, st.cfg.CommandConfig.Args...)
	// Prepare command
	cmd := exec.Command("qs", qsArgs...)

	// if there is something to pass to command stdin
	if st.cfg.CommandConfig.Stdin != "" {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return "", "", fmt.Errorf("failed to get stdin pipe: %w", err)
		}

		_, err = stdin.Write([]byte(st.cfg.CommandConfig.Stdin))
		if err != nil {
			return "", "", fmt.Errorf("failed to write to command stdin: %w", err)
		}

		stdin.Close()
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Dir = st.cloneRepoPath
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}

func (st *SystemTest) validateStdout(stdout string) error {
	// Check stdout if specified
	if st.cfg.ExpectedStdout != "" {
		if !strings.Contains(stdout, st.cfg.ExpectedStdout) {
			return fmt.Errorf("expected stdout to contain: %q, got: %q",
				st.cfg.ExpectedStdout, stdout)
		}
	}

	return nil
}

func (st *SystemTest) validateStderr(stderr string) error {
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
func (st *SystemTest) checkExpectations(re *RuntimeEnvironment) error {
	for _, expectation := range st.cfg.Expectations {
		if err := expectation.Check(re); err != nil {
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
	re, err := st.createTestEnvironment()
	if err != nil {
		return err
	}

	// Run the command
	stdout, stderr, err := st.runCommand()
	if err != nil {
		if err := st.validateStderr(stderr); err != nil {
			return err
		}

		return err
	}

	// Validate stdout
	if err := st.validateStdout(stdout); err != nil {
		return err
	}

	// Check expectations
	if err := st.checkExpectations(re); err != nil {
		return err
	}

	return st.cleanupTestEnvironment()
}

func (st *SystemTest) cleanupTestEnvironment() error {
	// Remove the cloned repository
	if err := os.RemoveAll(st.cloneRepoPath); err != nil {
		return fmt.Errorf("failed to remove cloned repository: %w", err)
	}

	// Optionally remove the upstream and fork repositories
	if st.cfg.UpstreamState != RemoteStateNull {
		if err := st.removeRepo(st.repoName, st.cfg.GHConfig.UpstreamAccount, st.cfg.GHConfig.UpstreamToken); err != nil {
			return fmt.Errorf("failed to remove upstream repository: %w", err)
		}
	}

	if st.cfg.ForkState != RemoteStateNull {
		if err := st.removeRepo(st.repoName, st.cfg.GHConfig.ForkAccount, st.cfg.GHConfig.ForkToken); err != nil {
			return fmt.Errorf("failed to remove fork repository: %w", err)
		}
	}

	return nil
}

// removeRepo removes a repository from GitHub
func (st *SystemTest) removeRepo(repoName, account, token string) error {
	cmd := exec.Command("gh", "repo", "delete",
		fmt.Sprintf("%s/%s", account, repoName),
		"--yes")

	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GITHUB_TOKEN=%s", token))

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete repository: %w\nOutput: %s", err, output)
	}

	return nil
}
