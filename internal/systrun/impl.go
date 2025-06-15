package systrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/gitcmds"
	contextPkg "github.com/untillpro/qs/internal/context"
	"io"
	netHttp "net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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
func (st *SystemTest) createTestEnvironment() error {
	st.ctx = context.WithValue(st.ctx, contextPkg.CtxKeyCloneRepoPath, st.cloneRepoPath)

	// Setup upstream repo if needed
	if st.cfg.UpstreamState != RemoteStateNull {
		upstreamRepoURL := buildRemoteURL(
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
		forkRepoURL := buildRemoteURL(
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
	// TODO: add check in loop with deadline instead of sleep
	time.Sleep(time.Millisecond * 2000)
	if err := st.cloneRepo(cloneURL, st.cloneRepoPath, authToken); err != nil {
		return err
	}

	// Configure remotes based on test scenario
	if err := st.configureRemotes(st.repoName); err != nil {
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

	time.Sleep(2 * time.Second)

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

	resp, err := netHttp.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		logger.Verbose("Invitation sent successfully.")
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

		acceptResp, err := netHttp.DefaultClient.Do(acceptReq)
		if err != nil {
			return err
		}
		defer acceptResp.Body.Close()

		if acceptResp.StatusCode == 204 {
			logger.Verbose("Accepted invitation ID %d\n", invite.ID)
		} else {
			fmt.Printf("Failed to accept invitation ID %d: %s\n", invite.ID, acceptResp.Status)
		}
		time.Sleep(2 * time.Second)
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
		})
		//err = repo.Push(&git.PushOptions{
		//	RemoteName: "origin",
		//	Auth:       &http.TokenAuth{Token: st.cfg.GHConfig.UpstreamToken},
		//	//Auth: &http.BasicAuth{
		//	//	Username: "x-access-token",
		//	//	Password: st.cfg.GHConfig.UpstreamToken,
		//	//},
		//})
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
				//Auth: &http.BasicAuth{
				//	Username: "x-access-token",
				//	Password: st.cfg.GHConfig.ForkToken,
				//},
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

func buildRemoteURL(account, token, repoName string, isUpstream bool) string {
	//return "https://" + account + ":" + token + "@github.com/" + account + "/" + repoName + ".git"
	remoteType := "upstream"
	if !isUpstream {
		remoteType = "origin"
	}

	return "git@github-" + remoteType + ":" + account + "/" + repoName + ".git"
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
		//_, err := repo.Remote("origin")
		//if err != nil {
		// Add origin remote pointing to fork
		err = repo.DeleteRemote("origin")
		if err != nil {
			return fmt.Errorf("failed to delete origin remote: %w", err)
		}

		forkURL := buildRemoteURL(
			st.cfg.GHConfig.ForkAccount,
			st.cfg.GHConfig.ForkToken,
			repoName,
			false,
		)
		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{forkURL},
		})
		if err != nil {
			return fmt.Errorf("failed to create origin remote: %w", err)
		}
		//}

		_, err = repo.Remote("upstream")
		if err != nil {
			// Add upstream remote
			upstreamURL := buildRemoteURL(
				st.cfg.GHConfig.UpstreamAccount,
				st.cfg.GHConfig.UpstreamToken,
				repoName,
				true,
			)
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
			upstreamURL := buildRemoteURL(
				st.cfg.GHConfig.UpstreamAccount,
				st.cfg.GHConfig.UpstreamToken,
				repoName,
				true,
			)
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
				config.RefSpec(fmt.Sprintf("refs/heads/dev:refs/heads/dev")),
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
	qsArgs = append(qsArgs, "-C", st.cloneRepoPath)

	// run the qs command
	st.ctx, err = st.qsExecRootCmd(st.ctx, qsArgs)

	_ = stderrWriter.Close()
	_ = stdoutWriter.Close()

	// Wait for all output to be captured
	wg.Wait()

	return
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

func (st *SystemTest) processSyncState() error {
	// Handle sync state based on configuration
	switch st.cfg.SyncState {
	case SyncStateSynchronized:
		return st.installSyncStateSynchronized()
	default:
		return fmt.Errorf("not supported yet sync state: %s", st.cfg.SyncState)
	}
}

// installSyncStateSynchronized installs the synchronized state for the dev branch
func (st *SystemTest) installSyncStateSynchronized() error {
	if st.cfg.SyncState != SyncStateSynchronized {
		return nil // No custom dev branch state needed
	}

	// Set the expected dev branch name that will be used later for PR
	createdGithubIssueURL := st.ctx.Value(contextPkg.CtxKeyCreatedGithubIssueURL).(string)
	if createdGithubIssueURL == "" {
		return errors.New("failed to determine github issue URL. Use ClipboardContentGithubIssue")
	}

	// Run "qs dev custom-branch-name" using st.runCommand
	stdout, stderr, err := st.runCommand(CommandConfig{
		Command: "dev",
		Stdin:   "y",
		Args:    []string{},
	})
	if err != nil {
		return fmt.Errorf("failed to run qs dev command: %w, stderr: %s", err, stderr)
	}

	logger.Verbose(stdout)

	// Create 3 commits with different files
	for i := 1; i <= 3; i++ {
		fileName := fmt.Sprintf("%d.txt", i)
		filePath := filepath.Join(st.cloneRepoPath, fileName)
		fileContent := fmt.Sprintf("Content of file %d", i)

		// Create the file
		if err := os.WriteFile(filePath, []byte(fileContent), 0644); err != nil {
			return fmt.Errorf("failed to create file %s: %w", fileName, err)
		}

		// Add file to git
		addCmd := exec.Command("git", "-C", st.cloneRepoPath, "add", fileName)
		if err := addCmd.Run(); err != nil {
			return fmt.Errorf("failed to git add %s: %w", fileName, err)
		}

		// Commit the file
		commitCmd := exec.Command("git", "-C", st.cloneRepoPath, "commit", "-m", fmt.Sprintf("Add %s", fileName))
		if err := commitCmd.Run(); err != nil {
			return fmt.Errorf("failed to commit %s: %w", fileName, err)
		}
	}

	devBranchName := gitcmds.GetCurrentBranchName(st.cloneRepoPath)

	// Push the dev branch to the remote
	pushCmd := exec.Command("git", "-C", st.cloneRepoPath, "push", "-u", "origin", devBranchName)
	pushOutput, err := pushCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push dev branch: %w, output: %s", err, pushOutput)
	}

	time.Sleep(2 * time.Second)

	return nil
}

// checkExpectations checks expectations after command execution
func (st *SystemTest) checkExpectations() error {
	for _, expectation := range st.cfg.Expectations {
		if err := expectation.Check(st.ctx); err != nil {
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

	if err := st.cleanupTestEnvironment(); err != nil {
		return err
	}

	return err
}

//	func (st *SystemTest) preRunCommand() error {
//		// authenticate with GitHub using the fork token
//		if err := utils.GhAuthLogin(st.cfg.GHConfig.ForkToken); err != nil {
//			return fmt.Errorf("failed to authenticate with GitHub: %w", err)
//		}
//
//		return nil
//	}
func (st *SystemTest) cleanupTestEnvironment() error {
	// Remove the cloned repository
	if err := os.RemoveAll(st.cloneRepoPath); err != nil {
		return fmt.Errorf("failed to remove cloned repository: %w", err)
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
