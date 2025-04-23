package systrun

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// AddCase adds a test case to the system test
func (st *SystemTest) AddCase(tc TestCase) {
	st.cases = append(st.cases, tc)
}

func (st *SystemTest) GetClonePath() string {
	return st.cloneDirPath
}

// Run executes all test cases in the system test
func (st *SystemTest) Run() error {
	if err := st.createEnv(); err != nil {
		return fmt.Errorf("failed to create environment: %w", err)
	}

	success := true
	for _, tc := range st.cases {
		st.t.Logf("Running test case: %s", tc.Name)

		stdout, stderr, err := st.runCommand(tc.Cmd, tc.Args...)
		if !tc.ErrorExpected {
			require.NoError(st.t, err)
		} else {
			require.Error(st.t, err)
		}

		if err != nil {
			st.t.Logf("Error running command: %v", err)
			success = false

			continue
		}

		if len(tc.Stdout) > 0 && !strings.Contains(stdout, tc.Stdout) {
			st.t.Logf("Expected stdout to contain: %s, got: %s", tc.Stdout, stdout)
			success = false

			continue
		}

		if len(tc.Stderr) > 0 && !strings.Contains(stderr, tc.Stderr) {
			st.t.Logf("Expected stderr to contain: %s, got: %s", tc.Stderr, stderr)
			success = false

			continue
		}

		if tc.CheckResults != nil {
			tc.CheckResults(st.t)
		}
	}

	if success {
		if err := st.deleteEnv(); err != nil {
			st.t.Logf("Failed to delete environment: %v", err)

			return err
		}
	} else {
		st.t.Logf("Test failed, keeping environment for debugging: %s", st.cfg.UpstreamRepoName)
	}

	return nil
}

// checkPrerequisites checks if all required tools are installed
func (st *SystemTest) checkPrerequisites() error {
	// Check if qs is installed
	_, err := exec.LookPath("qs")
	if err != nil {
		return fmt.Errorf("qs utility must be installed: %w", err)
	}

	// Check if git is installed
	_, err = exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git must be installed: %w", err)
	}

	// Check if gh is installed
	_, err = exec.LookPath("gh")
	if err != nil {
		return fmt.Errorf("GitHub CLI (gh) must be installed: %w", err)
	}

	// Check if gh is logged in
	cmd := exec.Command("gh", "auth", "status")
	err = cmd.Run()
	if err != nil {
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

	if err := st.createUpstreamRepo(); err != nil {
		return fmt.Errorf("failed to create main repo: %w", err)
	}

	return nil
}

// createUpstreamRepo creates the main repository
func (st *SystemTest) createUpstreamRepo() error {
	// Create GitHub repo using gh cli
	upstreamRepo := fmt.Sprintf("%s/%s", st.cfg.UpstreamGithubAccount, st.cfg.UpstreamRepoName)
	cmd := exec.Command("gh", "repo", "create", upstreamRepo, "--public", "--confirm")
	if st.cfg.GithubToken != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", EnvGithubToken, st.cfg.GithubToken))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create main repo: %v, output: %s", err, output)
	}

	// Clone the empty repo to a temporary directory
	tempDir, err := os.MkdirTemp("", "main-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Clone the main repo
	repoUrl := fmt.Sprintf("%s/%s/%s.git", githubURL, st.cfg.GithubAccount, st.cfg.UpstreamRepoName)
	if st.cfg.GithubOrg != "" {
		repoUrl = fmt.Sprintf("%s/%s/%s.git", githubURL, st.cfg.GithubOrg, st.cfg.UpstreamRepoName)
	}

	cmd = exec.Command("git", "clone", repoUrl, tempDir)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone main repo: %v, output: %s", err, output)
	}

	// Copy template files from testdata to the repo
	err = copyTemplateFiles("./internal/testdata/repo", tempDir)
	if err != nil {
		return fmt.Errorf("failed to copy template files: %w", err)
	}

	// Process go.mod.tmpl
	err = processGoModTemplate(tempDir, st.cfg.GithubAccount)
	if err != nil {
		return fmt.Errorf("failed to process go.mod.tmpl: %w", err)
	}

	// Add, commit and push files
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add files: %v, output: %s", err, output)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit files: %v, output: %s", err, output)
	}

	cmd = exec.Command("git", "push", "origin", "main")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push files: %v, output: %s", err, output)
	}

	return nil
}

// deleteEnv deletes the test environment
func (st *SystemTest) deleteEnv() error {
	// Delete fork repo
	repoName := fmt.Sprintf("%s/%s", st.cfg.GithubAccount, st.cfg.UpstreamRepoName)
	cmd := exec.Command("gh", "repo", "delete", repoName, "--yes")
	if st.cfg.GithubToken != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", EnvGithubToken, st.cfg.GithubToken))
	}

	_, err := cmd.CombinedOutput() // Ignore output
	if err != nil {
		st.t.Logf("Warning: failed to delete fork repo: %v", err)
		// Continue anyway
	}

	// Delete main repo
	repoName = fmt.Sprintf("%s/%s", st.cfg.GithubAccount, st.cfg.UpstreamRepoName)
	if st.cfg.GithubOrg != "" {
		repoName = fmt.Sprintf("%s/%s", st.cfg.GithubOrg, st.cfg.UpstreamRepoName)
	}

	cmd = exec.Command("gh", "repo", "delete", repoName, "--yes")
	if st.cfg.GithubToken != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", EnvGithubToken, st.cfg.GithubToken))
	}

	_, err = cmd.CombinedOutput() // Ignore output
	if err != nil {
		st.t.Logf("Warning: failed to delete main repo: %v", err)
		// Continue anyway
	}

	// Remove local clone
	if err := os.RemoveAll(st.cloneDirPath); err != nil {
		return fmt.Errorf("failed to remove clone directory: %w", err)
	}

	return nil
}

// runCommand runs a command in the clone repository
func (st *SystemTest) runCommand(command string, args ...string) (string, string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = st.cloneDirPath

	// Capture stdout and stderr
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// Helper functions

// copyTemplateFiles copies template files to the target directory
func copyTemplateFiles(srcDir, dstDir string) error {
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
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dstPath, err)
			}

			// Recursively copy files in directory
			if err := copyTemplateFiles(srcPath, dstPath); err != nil {
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
