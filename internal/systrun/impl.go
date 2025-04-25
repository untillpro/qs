package systrun

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// AddCase adds a test case to the system test
func (st *SystemTest) AddCase(tc TestCase) {
	st.cases = append(st.cases, tc)
}

// GetClonePath returns local path of the clone repository
func (st *SystemTest) GetClonePath() string {
	return filepath.Join(TestDataDir, st.cfg.UpstreamRepoName)
}

func (st *SystemTest) GetUpstreamRepoURL() string {
	return fmt.Sprintf("%s/%s/%s", GithubURL, st.cfg.UpstreamGithubAccount, st.cfg.UpstreamRepoName)
}

func (st *SystemTest) GetForkRepoURL() string {
	return fmt.Sprintf("%s/%s/%s", GithubURL, st.cfg.GithubAccount, st.cfg.UpstreamRepoName)
}

// Run executes all test cases in the system test
func (st *SystemTest) Run() {
	err := st.createEnv()
	require.NoErrorf(st.t, err, "failed to create environment: %v", err)
	st.t.Logf("Upstream repo URL: %s", st.GetUpstreamRepoURL())
	st.t.Logf("Fork repo URL: %s", st.GetForkRepoURL())
	st.t.Logf("Clone path: %s", st.GetClonePath())

	for _, tc := range st.cases {
		st.t.Logf("-------------------------------------")
		st.t.Logf("Running test case: %s", tc.Name)

		stdout, stderr, err := st.runCommand(tc.Cmd, tc.Args...)
		if !tc.ErrorExpected {
			require.NoErrorf(st.t, err, "Error running command: %v", err)
		} else {
			require.Error(st.t, err, "Error running command: %v", err)
		}

		if len(tc.Stdout) > 0 {
			require.Containsf(st.t, stdout, tc.Stdout, "Expected stdout %v, got %v", tc.Stdout, stdout)
		}

		if len(tc.Stderr) > 0 {
			require.Containsf(st.t, stderr, tc.Stderr, "Expected stderr %v, got %v", tc.Stderr, stderr)
		}

		if tc.CheckResults != nil {
			tc.CheckResults(st.t)
		}
	}

	if !st.cfg.KeepEnvAfterTest {
		err := st.deleteEnv()
		require.NoErrorf(st.t, err, "Failed to delete environment: %v", err)
	}
}

// checkPrerequisites checks if all required tools are installed
func (st *SystemTest) checkPrerequisites() error {
	// Check if qs is installed
	if _, err := exec.LookPath("qs"); err != nil {
		return fmt.Errorf("qs utility must be installed: %w", err)
	}

	// Check if git is installed
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git must be installed: %w", err)
	}

	// Check if gh is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) must be installed: %w", err)
	}

	cmd := exec.Command("gh", "auth", "login", "--with-token")
	cmd.Stdin = bytes.NewBufferString(st.cfg.GithubToken)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Auth failed: %v\n%s\n", err, string(output))
	}

	// Check if gh is logged in
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		return fmt.Errorf("GitHub CLI (gh) must be logged in: %w", err)
	}

	return nil
}

// createEnv creates the test environment
func (st *SystemTest) createEnv() error {
	// Check prerequisites
	if err := st.checkPrerequisites(); err != nil {
		return fmt.Errorf("prerequisites check failed: %w", err)
	}

	// Create an upstream repo
	if err := st.createUpstreamRepo(); err != nil {
		return fmt.Errorf("failed to create upstream repo: %w", err)
	}
	// Create .testdata dir If it doesn't exist
	return os.MkdirAll(TestDataDir, defaultDirPerms)
}

// createUpstreamRepo creates the upstream repository
func (st *SystemTest) createUpstreamRepo() error {
	// Create GitHub repo using gh cli
	upstreamRepo := fmt.Sprintf("%s/%s", st.cfg.UpstreamGithubAccount, st.cfg.UpstreamRepoName)
	cmd := exec.Command("gh", "repo", "create", upstreamRepo, "--private", "--confirm")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create upstream repo: %v, output: %s", err, output)
	}

	// Clone the empty repo to a temporary directory
	tempDirForCloneRepo, err := os.MkdirTemp("", "main-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDirForCloneRepo)
	}()

	// Clone the upstream repo
	repoUrl := fmt.Sprintf("%s/%s/%s.git", GithubURL, st.cfg.UpstreamGithubAccount, st.cfg.UpstreamRepoName)
	if st.cfg.GithubOrg != "" {
		repoUrl = fmt.Sprintf("%s/%s/%s.git", GithubURL, st.cfg.GithubOrg, st.cfg.UpstreamRepoName)
	}

	cmd = exec.Command("git", "clone", repoUrl, tempDirForCloneRepo)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone upstream repo: %v, output: %s", err, output)
	}

	// Copy template files from testdata to the repo
	err = copyFiles("./internal/testdata/repo", tempDirForCloneRepo)
	if err != nil {
		return fmt.Errorf("failed to copy template files: %w", err)
	}

	// Process go.mod.tmpl
	err = processGoModTemplate(tempDirForCloneRepo, st.cfg.GithubAccount)
	if err != nil {
		return fmt.Errorf("failed to process go.mod.tmpl: %w", err)
	}

	// Add, commit and push files
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tempDirForCloneRepo
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add files: %v, output: %s", err, output)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDirForCloneRepo
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit files: %v, output: %s", err, output)
	}

	cmd = exec.Command("git", "push", "origin", "main")
	cmd.Dir = tempDirForCloneRepo
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push files: %v, output: %s", err, output)
	}

	return nil
}

// deleteEnv deletes the test environment: Github repos and local clone
func (st *SystemTest) deleteEnv() error {
	// Delete fork repo
	forkRepoName := fmt.Sprintf("%s/%s", st.cfg.GithubAccount, st.cfg.UpstreamRepoName)
	cmd := exec.Command("gh", "repo", "delete", forkRepoName, "--yes")
	if _, err := cmd.CombinedOutput(); err != nil {
		st.t.Logf("Warning: failed to delete fork repo: %v", err)
		// Continue anyway
	}

	upstreamRepoName := fmt.Sprintf("%s/%s", st.cfg.UpstreamGithubAccount, st.cfg.UpstreamRepoName)
	if st.cfg.GithubOrg != "" {
		upstreamRepoName = fmt.Sprintf("%s/%s", st.cfg.GithubOrg, st.cfg.UpstreamRepoName)
	}

	if upstreamRepoName != forkRepoName {
		// Delete upstream repo
		cmd = exec.Command("gh", "repo", "delete", upstreamRepoName, "--yes")
		if st.cfg.GithubToken != "" {
			cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", EnvGithubToken, st.cfg.GithubToken))
		}

		if _, err := cmd.CombinedOutput(); err != nil {
			st.t.Logf("Warning: failed to delete upstream repo: %v", err)
		}
	}

	// Remove local clone
	if err := os.RemoveAll(st.GetClonePath()); err != nil {
		return fmt.Errorf("failed to remove clone directory: %w", err)
	}

	return nil
}

// runCommand runs a command in the clone repository
func (st *SystemTest) runCommand(command string, args ...string) (string, string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = TestDataDir

	// Capture stdout and stderr
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// Helper functions

// copyFiles copies repo files to the working directory
func copyFiles(srcDir, dstDir string) error {
	// Get list of template files
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	// Copy each file
	for _, entry := range entries {
		srcPath := fmt.Sprintf("%s/%s", srcDir, entry.Name())
		dstPath := fmt.Sprintf("%s/%s", dstDir, entry.Name())

		if entry.IsDir() {
			// Create directory if it doesn't exist
			if err := os.MkdirAll(dstPath, defaultDirPerms); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dstPath, err)
			}

			// Recursively copy files in directory
			if err := copyFiles(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			content, err := os.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", srcPath, err)
			}

			if err := os.WriteFile(dstPath, content, 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", dstPath, err)
			}
		}
	}

	return nil
}

// processGoModTemplate processes go.mod.tmpl to create go.mod
func processGoModTemplate(dir, ghAccount string) error {
	// Read template
	tmplPath := fmt.Sprintf("%s/go.mod.tmpl", dir)
	content, err := os.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to read go.mod.tmpl: %w", err)
	}

	// Replace placeholders
	modContent := string(content)
	modContent = strings.Replace(modContent, "GithubAccount", ghAccount, -1)
	modContent = strings.Replace(modContent, "UUID", uuid.New().String(), -1)

	// Write go.mod
	modPath := fmt.Sprintf("%s/go.mod", dir)
	if err := os.WriteFile(modPath, []byte(modContent), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}

	return nil
}
