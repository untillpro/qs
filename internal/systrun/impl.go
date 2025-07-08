package systrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	netHttp "net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/uuid"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/gitcmds"
	"github.com/untillpro/qs/internal/commands"
	contextPkg "github.com/untillpro/qs/internal/context"
	"github.com/untillpro/qs/internal/helper"
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
func (st *SystemTest) createTestEnvironment() error {
	st.ctx = context.WithValue(st.ctx, contextPkg.CtxKeyCloneRepoPath, st.cloneRepoPath)

	// Setup upstream repo if needed
	if st.cfg.UpstreamState != RemoteStateNull {
		upstreamRepoURL := gitcmds.BuildRemoteURL(
			st.cfg.GHConfig.UpstreamAccount,
			st.cfg.GHConfig.UpstreamToken,
			st.repoName,
			true,
		)
		if err := st.createUpstreamRepo(st.repoName, upstreamRepoURL); err != nil {
			return err
		}
	}

	// Setup fork repo if needed
	if st.cfg.ForkState != RemoteStateNull {
		forkRepoURL := gitcmds.BuildRemoteURL(
			st.cfg.GHConfig.ForkAccount,
			st.cfg.GHConfig.ForkToken,
			st.repoName,
			false,
		)
		if err := st.createForkRepo(st.repoName, forkRepoURL); err != nil {
			return err
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
		return fmt.Errorf("cannot create test environment: both upstream and fork repos are null")
	}

	// Need some time to ensure the repo is created
	if helper.IsTest() {
		helper.Delay()
	}

	if err := st.cloneRepo(cloneURL, st.cloneRepoPath, authToken); err != nil {
		return err
	}

	// Configure remotes based on test scenario
	if err := st.configureRemotes(st.cloneRepoPath, st.repoName); err != nil {
		return err
	}

	// Setup dev branch if needed
	if err := st.setupDevBranch(); err != nil {
		return err
	}

	if err := st.configureCollaboration(); err != nil {
		return err
	}

	if err := st.processClipboardContent(); err != nil {
		return fmt.Errorf("failed to process clipboard content: %w", err)
	}

	if err := st.processSyncState(); err != nil {
		return fmt.Errorf("failed to process sync state: %w", err)
	}

	if err := st.createAnotherClone(); err != nil {
		return fmt.Errorf("failed to create another clone: %w", err)
	}

	return nil
}

func (st *SystemTest) createAnotherClone() error {
	if !st.cfg.RunCommandFromAnotherClone {
		return nil
	}

	// get remotes from main clone
	remoteOriginURL, err := gitcmds.GetRemoteUrlByName(st.cloneRepoPath, "origin")
	if err != nil {
		return fmt.Errorf("failed to get oririn remote URL: %w", err)
	}

	remoteUpstreamURL, err := gitcmds.GetRemoteUrlByName(st.cloneRepoPath, "upstream")
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			remoteUpstreamURL = ""
		}

		return fmt.Errorf("failed to get upstream remote URL: %w", err)
	}

	// extract account, repo and token from remote url
	forkAccount, repo, forkToken, err := gitcmds.ParseGitRemoteURL(remoteOriginURL)
	if err != nil {
		return err
	}

	// extract account, repo and token from remote url
	upstreamAccount, repo, upstreamToken, err := gitcmds.ParseGitRemoteURL(remoteUpstreamURL)
	if err != nil {
		return err
	}

	// create temp dir for another clone
	tempPath, err := os.MkdirTemp("", "qs-test-clone-*")
	if err != nil {
		return fmt.Errorf("failed to create temp clone path: %w", err)
	}
	// set path to another clone
	st.anotherCloneRepoPath = filepath.Join(tempPath, st.repoName)
	// put path to the another clone to the context
	st.ctx = context.WithValue(st.ctx, contextPkg.CtxKeyAnotherCloneRepoPath, st.anotherCloneRepoPath)

	// clone  repo to the temp dir
	cloneCmd := exec.Command("git", "clone", remoteOriginURL)
	cloneCmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", forkToken))
	cloneCmd.Dir = tempPath

	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone repository: %w, output: %s", err, output)
	}

	// Step 3.1: Configure remotes in temp clone
	if err := gitcmds.CreateRemote(
		st.anotherCloneRepoPath,
		"upstream",
		upstreamAccount,
		upstreamToken,
		repo,
		true,
	); err != nil {
		return err
	}

	if err := gitcmds.CreateRemote(
		st.anotherCloneRepoPath,
		"origin",
		forkAccount,
		forkToken,
		repo,
		false,
	); err != nil {
		return err
	}

	remoteBranchName, ok := st.ctx.Value(contextPkg.CtxKeyDevBranchName).(string)
	if !ok {
		return fmt.Errorf("remote branch name not found in context")
	}

	if err := checkoutOnBranch(st.anotherCloneRepoPath, remoteBranchName); err != nil {
		return err
	}

	return nil
}

func (st *SystemTest) configureCollaboration() error {
	if err := inviteCollaborator(
		st.cfg.GHConfig.UpstreamAccount,
		st.repoName,
		st.cfg.GHConfig.ForkAccount,
		st.cfg.GHConfig.UpstreamToken,
	); err != nil {
		return err
	}

	if helper.IsTest() {
		helper.Delay()
	}

	if err := acceptPendingInvitations(st.cfg.GHConfig.ForkToken); err != nil {
		return err
	}

	return nil
}

func inviteCollaborator(owner, repo, username, token string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s", owner, repo, username)

	// Request body
	body := map[string]string{
		"permission": "push", // Or "pull", "admin"
	}
	jsonBody, _ := json.Marshal(body)

	req, err := netHttp.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	// TODO: add retry logic
	resp, err := netHttp.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		logger.Error("Invitation sent successfully.")
	} else {
		return fmt.Errorf("failed to invite: status %s", resp.Status)
	}
	return nil
}

func acceptPendingInvitations(token string) error {
	url := "https://api.github.com/user/repository_invitations"

	req, _ := netHttp.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := netHttp.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var invitations []struct {
		ID int `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&invitations); err != nil {
		return err
	}

	for _, invite := range invitations {
		acceptURL := fmt.Sprintf("https://api.github.com/user/repository_invitations/%d", invite.ID)
		acceptReq, _ := netHttp.NewRequest("PATCH", acceptURL, nil)
		acceptReq.Header.Set("Authorization", "token "+token)
		acceptReq.Header.Set("Accept", "application/vnd.github+json")

		// TODO: add retry logic
		acceptResp, err := netHttp.DefaultClient.Do(acceptReq)
		if err != nil {
			return err
		}
		defer acceptResp.Body.Close()

		if acceptResp.StatusCode == 204 {
			logger.Error("Accepted invitation ID %d\n", invite.ID)
		} else {
			fmt.Printf("Failed to accept invitation ID %d: %s\n", invite.ID, acceptResp.Status)
		}

		if helper.IsTest() {
			helper.Delay()
		}
	}

	return nil
}

func (st *SystemTest) processClipboardContent() error {
	var (
		clipboardContent string
		err              error
	)

	switch st.cfg.ClipboardContent {
	case ClipboardContentEmpty:
		return nil
	case ClipboardContentCustom:
		clipboardContent = st.getCustomClipboardContent()

		st.ctx = context.WithValue(st.ctx, contextPkg.CtxKeyCustomBranchName, clipboardContent+"-dev")
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

		st.ctx = context.WithValue(st.ctx, contextPkg.CtxKeyCreatedGithubIssueURL, clipboardContent)
	case ClipboardContentJiraTicket:
		clipboardContent = os.Getenv("JIRA_TICKET_URL")
		if clipboardContent == "" {
			return errors.New("JIRA_TICKET_URL environment variable not set, skipping test")
		}

		jiraTicketID, ok := commands.GetJiraTicketIDFromArgs(clipboardContent)
		if !ok {
			return fmt.Errorf("invalid JIRA ticket URL: %s", clipboardContent)
		}

		st.ctx = context.WithValue(st.ctx, contextPkg.CtxKeyBranchPrefix, jiraTicketID)
	default:
		return fmt.Errorf("unknown clipboard content type: %s", st.cfg.ClipboardContent)
	}
	// put clipboard value to context
	st.ctx = context.WithValue(st.ctx, contextPkg.CtxKeyClipboard, clipboardContent)

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

func checkoutOnBranch(wd, branchName string) error {
	if branchName == "" {
		return nil
	}

	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = wd
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout on %s branch: %w, output: %s", branchName, err, output)
	}

	return nil
}

// createUpstreamRepo creates the upstream repository
func (st *SystemTest) createUpstreamRepo(repoName, repoURL string) error {
	// GitHub Authentication and repo creation with retry
	err := helper.RetryWithMaxAttempts(func() error {
		cmd := exec.Command("gh", "repo", "create",
			fmt.Sprintf("%s/%s", st.cfg.GHConfig.UpstreamAccount, repoName),
			"--public")

		cmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", st.cfg.GHConfig.UpstreamToken))

		if output, createErr := cmd.CombinedOutput(); createErr != nil {
			return fmt.Errorf("failed to create upstream repo: %w\nOutput: %s", createErr, output)
		}
		return nil
	}, 3) // Retry up to 3 times for repo creation
	if err != nil {
		return err
	}

	// Verify repository was created and is accessible with retry
	err = helper.RetryWithMaxAttempts(func() error {
		return helper.VerifyGitHubRepoExists(st.cfg.GHConfig.UpstreamAccount, repoName, st.cfg.GHConfig.UpstreamToken)
	}, 5) // Retry up to 5 times for verification (GitHub eventual consistency)
	if err != nil {
		return fmt.Errorf("upstream repository verification failed: %w", err)
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

		// Push changes with retry
		err = helper.RetryWithMaxAttempts(func() error {
			return repo.Push(&git.PushOptions{
				RemoteName: "origin",
				Auth: &http.BasicAuth{
					Username: "x-access-token",
					Password: st.cfg.GHConfig.UpstreamToken,
				},
			})
		}, 3) // Retry up to 3 times for pushing changes
		if err != nil {
			return fmt.Errorf("failed to push changes: %w", err)
		}
	}

	return nil
}

// createForkRepo creates or configures the fork repository
func (st *SystemTest) createForkRepo(repoName, repoURL string) error {
	if st.cfg.UpstreamState != RemoteStateNull {
		// Fork the upstream repo with retry
		err := helper.RetryWithMaxAttempts(func() error {
			cmd := exec.Command(
				"gh",
				"repo",
				"fork",
				fmt.Sprintf("%s/%s", st.cfg.GHConfig.UpstreamAccount, repoName),
				"--clone=false",
			)

			cmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", st.cfg.GHConfig.ForkToken))

			if output, forkErr := cmd.CombinedOutput(); forkErr != nil {
				return fmt.Errorf("failed to fork upstream repo: %w\nOutput: %s", forkErr, output)
			}
			return nil
		}, 3) // Retry up to 3 times for repo fork
		if err != nil {
			return err
		}

		// Verify fork was created and is accessible with retry
		err = helper.RetryWithMaxAttempts(func() error {
			return helper.VerifyGitHubRepoExists(st.cfg.GHConfig.ForkAccount, repoName, st.cfg.GHConfig.ForkToken)
		}, 5) // Retry up to 5 times for verification (GitHub eventual consistency)
		if err != nil {
			return fmt.Errorf("fork repository verification failed: %w", err)
		}
	} else {
		// Create an independent repo with retry
		err := helper.RetryWithMaxAttempts(func() error {
			cmd := exec.Command(
				"gh",
				"repo",
				"create",
				fmt.Sprintf("%s/%s", st.cfg.GHConfig.ForkAccount, repoName),
				"--public",
			)

			cmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", st.cfg.GHConfig.ForkToken))

			if output, createErr := cmd.CombinedOutput(); createErr != nil {
				return fmt.Errorf("failed to create fork repo: %w\nOutput: %s", createErr, output)
			}
			return nil
		}, 3) // Retry up to 3 times for repo creation
		if err != nil {
			return err
		}

		// Verify repository was created and is accessible with retry
		err = helper.RetryWithMaxAttempts(func() error {
			return helper.VerifyGitHubRepoExists(st.cfg.GHConfig.ForkAccount, repoName, st.cfg.GHConfig.ForkToken)
		}, 5) // Retry up to 5 times for verification (GitHub eventual consistency)
		if err != nil {
			return fmt.Errorf("fork repository verification failed: %w", err)
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

			// Push changes with retry
			err = helper.RetryWithMaxAttempts(func() error {
				return repo.Push(&git.PushOptions{
					RemoteName: "origin",
					Auth: &http.BasicAuth{
						Username: "x-access-token",
						Password: st.cfg.GHConfig.ForkToken,
					},
				})
			}, 3) // Retry up to 3 times for pushing changes
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

// configureRemotes sets up the remote configuration in the cloned repository
func (st *SystemTest) configureRemotes(wd, repoName string) error {
	// Configure remotes based on test scenario
	switch {
	case st.cfg.ForkState != RemoteStateNull && st.cfg.UpstreamState != RemoteStateNull:
		// Both upstream and fork exist, configure both remotes
		if err := gitcmds.CreateRemote(
			wd,
			"origin",
			st.cfg.GHConfig.ForkAccount,
			st.cfg.GHConfig.ForkToken,
			repoName,
			false,
		); err != nil {
			return err
		}

		if err := gitcmds.CreateRemote(
			wd,
			"upstream",
			st.cfg.GHConfig.UpstreamAccount,
			st.cfg.GHConfig.UpstreamToken,
			repoName,
			true,
		); err != nil {
			return err
		}

	case st.cfg.ForkState == RemoteStateNull && st.cfg.UpstreamState != RemoteStateNull:
		// Only upstream exists, make sure origin points to upstream
		if err := gitcmds.CreateRemote(
			wd,
			"origin",
			st.cfg.GHConfig.UpstreamAccount,
			st.cfg.GHConfig.UpstreamToken,
			repoName,
			true,
		); err != nil {
			return err
		}
	default:
		return errors.New("incorrect remote state configuration")
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
	devBranchName := "dev-dev"
	branchRef := plumbing.NewBranchReferenceName(devBranchName)
	ref := plumbing.NewHashReference(branchRef, headRef.Hash())
	if err := repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("failed to create dev branch: %w", err)
	}

	// Checkout the dev branch if it should be current
	if st.cfg.DevBranchState == DevBranchStateExistsAndCheckedOut {
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
				config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", devBranchName, devBranchName)),
			},
		})
		if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return fmt.Errorf("failed to push dev branch to fork: %w", err)
		}
	}

	return nil
}

// runCommand executes the specified qs command and captures stdout and stderr
func (st *SystemTest) runCommand(cmdCfg CommandConfig) (stdout string, stderr string, err error) {
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		// notestdept
		return
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		// notestdept
		return
	}

	// Create pipe for stdin if needed
	var stdinReader, stdinWriter *os.File
	if cmdCfg.Stdin != "" {
		stdinReader, stdinWriter, err = os.Pipe()
		if err != nil {
			return "", "", fmt.Errorf("failed to create stdin pipe: %w", err)
		}
	}

	origStdin := os.Stdin
	origStdout := os.Stdout
	origStderr := os.Stderr

	// Set up new stdin if needed
	if stdinReader != nil {
		os.Stdin = stdinReader
		defer func() { os.Stdin = origStdin }()
	}

	os.Stdout = stdoutWriter
	defer func() { os.Stdout = origStdout }()

	os.Stderr = stderrWriter
	defer func() { os.Stderr = origStderr }()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		var b bytes.Buffer
		defer wg.Done()
		_, _ = io.Copy(&b, stdoutReader)
		stdout = b.String()
	}()
	wg.Add(1)
	go func() {
		var b bytes.Buffer
		defer wg.Done()
		_, _ = io.Copy(&b, stderrReader)
		stderr = b.String()
	}()

	// Handle stdin if needed
	if cmdCfg.Stdin != "" {
		go func() {
			_, _ = stdinWriter.WriteString(cmdCfg.Stdin)
			_ = stdinWriter.Close()
		}()
	}

	// Prepare the qs command arguments
	qsArgs := make([]string, 0, len(cmdCfg.Args)+1)
	qsArgs = append(qsArgs, "qs")
	qsArgs = append(qsArgs, cmdCfg.Command)
	qsArgs = append(qsArgs, cmdCfg.Args...)

	runDir := st.cloneRepoPath
	if st.cfg.RunCommandFromAnotherClone && st.anotherCloneRepoPath != "" {
		runDir = st.anotherCloneRepoPath
	}
	qsArgs = append(qsArgs, "-C", runDir)

	// run the qs command
	st.ctx, err = st.qsExecRootCmd(st.ctx, qsArgs)

	_ = stderrWriter.Close()
	_ = stdoutWriter.Close()

	// Wait for all output to be captured
	wg.Wait()

	return
}

func (st *SystemTest) validateStdout(stdout string) error {
	_, _ = fmt.Fprintln(os.Stdout, stdout)

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
	_, _ = fmt.Fprintln(os.Stderr, stderr)

	// Check stderr if specified
	if st.cfg.ExpectedStderr != "" {
		if !strings.Contains(stderr, st.cfg.ExpectedStderr) {
			return fmt.Errorf("expected stderr to contain: %q, got: %q",
				st.cfg.ExpectedStderr, stderr)
		}
	}

	return nil
}

func (st *SystemTest) processSyncState() error {
	// Handle sync state based on configuration
	switch st.cfg.SyncState {
	case SyncStateUnspecified:
		return nil
	case SyncStateUncommitedChangesInClone:
		return st.setSyncState(false, true, true, false, "")
	case SyncStateSynchronized:
		return st.setSyncState(true, true, true, false, "")
	case SyncStateCloneChanged:
		return st.setSyncState(true, true, false, false, "")
	case SyncStateForkChanged:
		return st.setSyncState(false, false, false, true, headerOfFilesInAnotherClone, 4)
	case SyncStateBothChanged:
		return st.setSyncState(true, true, false, true, headerOfFilesInAnotherClone, 4)
	case SyncStateBothChangedConflict:
		return st.setSyncState(true, true, false, true, headerOfFilesInAnotherClone, 3)
	default:
		return fmt.Errorf("not supported yet sync state: %s", st.cfg.SyncState)
	}
}

// setSyncState installs the synchronized state for the dev branch
// Parameters:
// - needToCommit: if true then create commits in clone repo
// - needChangeClone: if true then add files into clone repo
// - needSync: if true then push changes from clone to remote
// - needChangeFork: if true then push new commits from another clone
// - headerOfFilesInFork: optional header to be added to each file in fork
// - idFilesInFork: list of file IDs to create in fork (e.g., 1, 2, 3 for 1.txt, 2.txt, 3.txt)
func (st *SystemTest) setSyncState(
	needToCommit bool,
	needChangeClone bool,
	needSync bool,
	needChangeFork bool,
	headerOfFilesInFork string,
	idFilesInFork ...int,
) error {
	if st.cfg.SyncState == SyncStateUnspecified || st.cfg.SyncState == SyncStateDoesntTrackOrigin {
		return errors.New("sync state is not supported")
	}

	// Set the expected dev branch name that will be used later for PR
	createdGithubIssueURL := st.ctx.Value(contextPkg.CtxKeyCreatedGithubIssueURL).(string)
	if createdGithubIssueURL == "" {
		return errors.New("failed to determine github issue URL. Use ClipboardContentGithubIssue")
	}

	// authenticate with GitHub using the fork token
	if err := os.Setenv("GITHUB_TOKEN", st.cfg.GHConfig.ForkToken); err != nil {
		return err
	}

	stdout, stderr, err := st.runCommand(CommandConfig{
		Command: "dev",
		Stdin:   "y",
	})
	if err != nil {
		return fmt.Errorf("failed to run qs dev command: %w, stderr: %s", err, stderr)
	}

	logger.Error(stdout)

	if needChangeClone {
		// Create 3 commits with different files
		if err := commitFiles(st.cloneRepoPath, needToCommit, "", 1, 2, 3); err != nil {
			return err
		}
	}

	if needSync {
		devBranchName := gitcmds.GetCurrentBranchName(st.cloneRepoPath)
		// Push the dev branch to the remote with retry logic
		err := helper.RetryWithMaxAttempts(func() error {
			pushCmd := exec.Command("git", "-C", st.cloneRepoPath, "push", "-u", "origin", devBranchName)
			pushCmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", st.cfg.GHConfig.ForkToken))
			pushOutput, pushErr := pushCmd.CombinedOutput()
			if pushErr != nil {
				return fmt.Errorf("failed to push dev branch: %w, output: %s", pushErr, pushOutput)
			}
			return nil
		}, 3) // Retry up to 3 times for pushing dev branch
		if err != nil {
			return err
		}
	}

	if needChangeFork {
		remoteURL, err := gitcmds.GetRemoteUrlByName(st.cloneRepoPath, "origin")
		if err != nil {
			return fmt.Errorf("failed to get remote URL: %w", err)
		}

		devBranchName := st.ctx.Value(contextPkg.CtxKeyDevBranchName).(string)
		if devBranchName == "" {
			return errors.New("failed to determine dev branch name")
		}

		// push changes from another clone
		if err := st.pushFromAnotherClone(remoteURL, devBranchName, headerOfFilesInFork, idFilesInFork...); err != nil {
			return err
		}

	}

	if helper.IsTest() {
		helper.Delay()
	}

	return nil
}

// commitFiles creates files in the cloned repository and commits them
// Parameters:
// - headerOfFiles: optional header to be added to each file
// - idFiles: list of file IDs to create (e.g., 1, 2, 3 for 1.txt, 2.txt, 3.txt)
func commitFiles(wd string, needToCommit bool, headerOfFiles string, idFiles ...int) error {
	// Create 3 commits with different files
	for _, id := range idFiles {
		fileName := fmt.Sprintf("%d.txt", id)
		filePath := filepath.Join(wd, fileName)
		fileContent := strings.Builder{}
		if headerOfFiles != "" {
			fileContent.WriteString(headerOfFiles + "\n")
		}
		fileContent.WriteString(fmt.Sprintf("Content of file %d", id))

		// Create the file
		if err := os.WriteFile(filePath, []byte(fileContent.String()), 0644); err != nil {
			return fmt.Errorf("failed to create file %s: %w", fileName, err)
		}

		// Add file to git
		addCmd := exec.Command("git", "-C", wd, "add", fileName)
		if err := addCmd.Run(); err != nil {
			return fmt.Errorf("failed to git add %s: %w", fileName, err)
		}

		if needToCommit {
			// Commit the file
			commitCmd := exec.Command("git", "-C", wd, "commit", "-m", fmt.Sprintf("Add %s", fileName))
			if err := commitCmd.Run(); err != nil {
				return fmt.Errorf("failed to commit %s: %w", fileName, err)
			}
		}
	}

	return nil
}

// pushFromAnotherClone pushes changes from another clone of the repository
// This simulates a scenario where changes are made in a different clone and pushed to the remote.
// Parameters:
// - originRemoteURL: the remote URL of the original repository
// - branchName: the name of the branch to push changes to
// - headOfFiles: optional header to be added to each file
// - idFiles: list of file IDs to create (e.g., 1, 2, 3 for 1.txt, 2.txt, 3.txt)
func (st *SystemTest) pushFromAnotherClone(originRemoteURL, branchName, headOfFiles string, idFiles ...int) error {
	// 1. Clone the repository to temp path
	// 2. Pull the latest changes from the remote
	// 3. Commit to the branchName
	// 4. Push the branch to the remote

	// Step 1: Get account and repo name from originRemoteURL
	account, repo, token, err := gitcmds.ParseGitRemoteURL(originRemoteURL)
	if err != nil {
		return err
	}

	// Step 2: Create temp path for the clone
	tempPath, err := os.MkdirTemp("", "qs-test-clone-*")
	if err != nil {
		return fmt.Errorf("failed to create temp clone path: %w", err)
	}

	defer os.RemoveAll(tempPath)

	// Step 3: Clone the repository in the temp path
	tempClonePath := filepath.Join(tempPath, repo)
	cloneCmd := exec.Command("git", "clone", originRemoteURL)
	cloneCmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", token))
	cloneCmd.Dir = tempPath

	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone repository: %w, output: %s", err, output)
	}

	// Step 4: Create the remote in the cloned repository
	if err := gitcmds.CreateRemote(
		tempClonePath,
		"origin",
		account,
		token,
		repo,
		false,
	); err != nil {
		return err
	}

	// Step 4.1: Fetch the dev branch from origin
	fetchCmd := exec.Command("git", "fetch", "origin", branchName)
	fetchCmd.Dir = tempClonePath
	fetchCmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", token))
	if output, err := fetchCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fetch %s branch from origin: %w, output: %s", branchName, err, output)
	}

	// Step 5: Change to the cloned repository directory
	if err := checkoutOnBranch(tempClonePath, branchName); err != nil {
		return err
	}

	// Step 6: Create files in the cloned repository
	if err := commitFiles(tempClonePath, true, headOfFiles, idFiles...); err != nil {
		return err
	}

	// Step 7: Push the branch to the remote
	pushCmd := exec.Command("git", "-C", tempClonePath, "push", "origin", branchName)
	pushCmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", token))
	if output, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push branch %s: %w, output: %s", branchName, err, output)
	}

	return nil
}

// checkExpectations checks expectations after command execution
func (st *SystemTest) checkExpectations() error {
	for _, expectation := range st.cfg.Expectations {
		if err := expectation(st.ctx); err != nil {
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
	if err := st.checkPrerequisites(); err != nil {
		return err
	}

	if err := st.createTestEnvironment(); err != nil {
		return err
	}

	// authenticate with GitHub using the fork token
	if err := os.Setenv("GITHUB_TOKEN", st.cfg.GHConfig.ForkToken); err != nil {
		return err
	}

	stdout, stderr, err := st.runCommand(st.cfg.CommandConfig)
	if err != nil {
		if err := st.validateStderr(stderr); err != nil {
			return err
		}
	}

	if err := st.validateStdout(stdout); err != nil {
		return err
	}

	if err := st.checkExpectations(); err != nil {
		return err
	}

	//if err := st.cleanupTestEnvironment(); err != nil {
	//	return err
	//}

	return err
}

// cleanupTestEnvironment removes all created resources: cloned repo, upstream repo, fork repo
func (st *SystemTest) cleanupTestEnvironment() error {
	// Remove the cloned repository
	if err := os.RemoveAll(st.cloneRepoPath); err != nil {
		return fmt.Errorf("failed to remove cloned repository: %w", err)
	}

	// Remove the another cloned repository
	if st.anotherCloneRepoPath != "" {
		if err := os.RemoveAll(st.anotherCloneRepoPath); err != nil {
			return fmt.Errorf("failed to remove another cloned repository: %w", err)
		}
	}

	// Optionally remove the upstream and fork repositories
	if st.cfg.UpstreamState == RemoteStateOK {
		if err := st.removeRepo(st.repoName, st.cfg.GHConfig.UpstreamAccount, st.cfg.GHConfig.UpstreamToken); err != nil {
			return fmt.Errorf("failed to remove upstream repository: %w", err)
		}
	}

	if st.cfg.ForkState == RemoteStateOK {
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
