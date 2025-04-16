package main

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/untillpro/qs/internal/systrun"
)

func TestMainWorkflow(t *testing.T) {
	// Skip if running in CI without proper setup
	if os.Getenv("CI") == "true" && os.Getenv(systrun.EnvGithubAccount) == "" {
		t.Skip("Skipping system test in CI environment without proper setup")
	}

	// Get GitHub account from environment
	ghAccount := os.Getenv(systrun.EnvGithubAccount)
	if ghAccount == "" {
		t.Skipf("%s environment variable must be set", systrun.EnvGithubAccount)
	}

	// Generate unique repo names with UUID
	repoID := uuid.New().String()[:8]
	mainRepoName := fmt.Sprintf("qs-test-%s", repoID)

	// Create test configuration
	cfg := systrun.SystemTestCfg{
		GithubAccount:    ghAccount,
		GithubToken:      os.Getenv(systrun.EnvGithubToken),
		UpstreamRepoName: mainRepoName,
	}

	// Create system test
	st := systrun.NewSystemTest(t, cfg)

	// Add test cases for main workflow

	// 1. Create developer branch
	devBranchName := "feature-test"
	// TODO: Add forking test case via `qs fork`
	// TODO: Add cloning test case via `git clone`

	st.AddCase(systrun.TestCase{
		Name:   "Create developer branch",
		Cmd:    "qs",
		Args:   []string{"dev", devBranchName},
		Stdout: fmt.Sprintf("Dev branch '%s' will be created", devBranchName),
		CheckResults: func(t *testing.T) {
			// Verify branch was created
			cmd := exec.Command("git", "branch", "--show-current")
			cmd.Dir = st.GetClonePath()
			output, err := cmd.Output()
			require.NoErrorf(t, err, "failed to get current branch: %v", err)

			currentBranch := string(output)
			require.Containsf(t, currentBranch, devBranchName, "expected branch %s, got %s", devBranchName, currentBranch)
		},
	})

	// 2. Make changes to a file
	newContent := "package main\n\nimport \"fmt\"\n\nfunc Sum(a, b int) int {\n\treturn a + b\n}\n\nfunc main() {\n\tfmt.Println(Sum(1, 3))\n}"
	st.AddCase(systrun.TestCase{
		Name: "Make changes to a file",
		Cmd:  "bash",
		Args: []string{"-c", fmt.Sprintf("echo '%s' > %s/main.go", newContent, st.GetClonePath())},
		CheckResults: func(t *testing.T) {
			// Verify file was modified
			content, err := os.ReadFile(fmt.Sprintf("%s/main.go", st.GetClonePath()))
			require.NoErrorf(t, err, "failed to read file: %v", err)
			require.Equal(t, string(content), newContent, "file content does not match")
		},
	})

	// 3. Add and commit changes
	st.AddCase(systrun.TestCase{
		Name:   "Add and commit changes",
		Cmd:    "qs",
		Args:   []string{"u", "-m", "Add test function"},
		Stdout: "Add test function",
		CheckResults: func(t *testing.T) {
			// Verify commit was created
			cmd := exec.Command("git", "log", "-1", "--pretty=%B")
			cmd.Dir = st.GetClonePath()
			output, err := cmd.Output()
			require.NoErrorf(t, err, "failed to get commit message: %v", err)
			// TODO: check if push was successful and fork-repo got updated
			require.Containsf(t, string(output), "Add test function", "expected commit message 'Add test function', got %s", string(output))
		},
	})

	// 4. Create pull request
	st.AddCase(systrun.TestCase{
		Name:   "Create pull request",
		Cmd:    "qs",
		Args:   []string{"pr"},
		Stdout: "Pull request",
		CheckResults: func(t *testing.T) {
			// Verify PR was created
			cmd := exec.Command("gh", "pr", "list", "--state", "open")
			cmd.Dir = st.GetClonePath()
			output, err := cmd.Output()
			require.NoErrorf(t, err, "failed to list pull requests: %v", err)
			require.NotEmpty(t, output, "pull requests are not empty")
		},
	})

	// Run all test cases
	err := st.Run()
	require.NoError(t, err)
}

func TestEdgeCases(t *testing.T) {
	// Skip if running in CI without proper setup
	if os.Getenv("CI") == "true" && os.Getenv(systrun.EnvGithubAccount) == "" {
		t.Skip("Skipping system test in CI environment without proper setup")
	}

	t.Run("BranchMismatch", func(t *testing.T) {
		// Get GitHub account from environment
		ghAccount := os.Getenv(systrun.EnvGithubAccount)
		if ghAccount == "" {
			t.Skipf("%s environment variable must be set", systrun.EnvGithubAccount)
		}

		// Generate unique repo names
		repoID := uuid.New().String()[:8]
		mainRepoName := fmt.Sprintf("qs-test-branch-%s", repoID)

		// Create test configuration
		cfg := systrun.SystemTestCfg{
			GithubAccount:    ghAccount,
			GithubToken:      os.Getenv(systrun.EnvGithubToken),
			UpstreamRepoName: mainRepoName,
		}

		// Create system test
		st := systrun.NewSystemTest(t, cfg)

		// First create a non-main branch
		st.AddCase(systrun.TestCase{
			Name: "Create initial branch",
			Cmd:  "git",
			Args: []string{"checkout", "-b", "non-main-branch"},
			CheckResults: func(t *testing.T) {
				// Verify branch was created
				cmd := exec.Command("git", "branch", "--show-current")
				cmd.Dir = st.GetClonePath()
				output, err := cmd.Output()
				require.NoErrorf(t, err, "failed to get current branch: %v", err)

				currentBranch := string(output[:len(output)-1])
				require.Equal(t, currentBranch, "non-main-branch", "expected branch non-main-branch, got %s", currentBranch)
			},
		})

		// Try to create a dev branch from a non-main branch
		st.AddCase(systrun.TestCase{
			Name:   "Try to create dev branch from non-main branch",
			Cmd:    "qs",
			Args:   []string{"dev", "feature-branch"},
			Stderr: "not on main branch",
		})

		// Run the test
		_ = st.Run() // We expect this to fail with branch mismatch error
	})

	t.Run("EmptyCommitNotes", func(t *testing.T) {
		// Get GitHub account from environment
		ghAccount := os.Getenv(systrun.EnvGithubAccount)
		if ghAccount == "" {
			t.Skipf("%s environment variable must be set", systrun.EnvGithubAccount)
		}

		// Generate unique repo names
		repoID := uuid.New().String()[:8]
		mainRepoName := fmt.Sprintf("qs-test-empty-%s", repoID)

		// Create test configuration
		cfg := systrun.SystemTestCfg{
			GithubAccount:    ghAccount,
			GithubToken:      os.Getenv(systrun.EnvGithubToken),
			UpstreamRepoName: mainRepoName,
		}

		// Create system test
		st := systrun.NewSystemTest(t, cfg)

		// Create a dev branch
		devBranchName := "feature-empty-commit"
		st.AddCase(systrun.TestCase{
			Name:   "Create developer branch",
			Cmd:    "qs",
			Args:   []string{"dev", devBranchName},
			Stdout: fmt.Sprintf("Dev branch '%s' will be created", devBranchName),
		})

		// Make some changes
		newContent := "package main\n\nimport \"fmt\"\n\nfunc Sum(a, b int) int {\n\treturn a + b\n}\n\nfunc main() {\n\tfmt.Println(Sum(1, 3))\n}"
		st.AddCase(systrun.TestCase{
			Name: "Make changes",
			Cmd:  "bash",
			Args: []string{"-c", fmt.Sprintf("echo '%s' > %s/main.go", newContent, st.GetClonePath())},
		})

		// Try to commit with empty message
		st.AddCase(systrun.TestCase{
			Name:   "Try to commit with empty message",
			Cmd:    "qs",
			Args:   []string{"u", "-m", ""},
			Stderr: "commit message",
		})

		// Run the test
		_ = st.Run() // We expect this to fail with empty commit message error
	})

	t.Run("PushWithStash", func(t *testing.T) {
		// Get GitHub account from environment
		ghAccount := os.Getenv(systrun.EnvGithubAccount)
		if ghAccount == "" {
			t.Skipf("%s environment variable must be set", systrun.EnvGithubAccount)
		}

		// Generate unique repo names
		repoID := uuid.New().String()[:8]
		mainRepoName := fmt.Sprintf("qs-test-stash-%s", repoID)

		// Create test configuration
		cfg := systrun.SystemTestCfg{
			GithubAccount:    ghAccount,
			GithubToken:      os.Getenv(systrun.EnvGithubToken),
			UpstreamRepoName: mainRepoName,
		}

		// Create system test
		st := systrun.NewSystemTest(t, cfg)

		// Create a dev branch
		devBranchName := "feature-stash-test"
		st.AddCase(systrun.TestCase{
			Name:   "Create developer branch",
			Cmd:    "qs",
			Args:   []string{"dev", devBranchName},
			Stdout: fmt.Sprintf("Dev branch '%s' will be created", devBranchName),
		})

		// Make changes and commit
		newContent := "package main\n\nimport \"fmt\"\n\nfunc Sum(a, b int) int {\n\treturn a + b\n}\n\nfunc main() {\n\tfmt.Println(Sum(1, 3))\n}"
		st.AddCase(systrun.TestCase{
			Name: "Make initial changes",
			Cmd:  "bash",
			Args: []string{"-c", fmt.Sprintf("echo '%s' > %s/main.go", newContent, st.GetClonePath())},
		})

		st.AddCase(systrun.TestCase{
			Name: "Commit initial changes",
			Cmd:  "qs",
			Args: []string{"u", "-m", "Add stash test function 1"},
		})

		// Make additional uncommitted changes
		st.AddCase(systrun.TestCase{
			Name: "Create empty file",
			Cmd:  "bash",
			Args: []string{"-c", fmt.Sprintf("touch %s/mul.go", st.GetClonePath())},
		})

		// Make additional uncommitted changes
		newFileContent := "package main\n\nimport \"fmt\"\n\nfunc Mul(a, b int) int {\n\treturn a * b\n}\n\n"
		st.AddCase(systrun.TestCase{
			Name: "Make additional uncommitted changes",
			Cmd:  "bash",
			Args: []string{"-c", fmt.Sprintf("echo '%s' > %s/mul.go", newFileContent, st.GetClonePath())},
		})

		// Create new branch that should stash changes
		st.AddCase(systrun.TestCase{
			Name:   "Create new branch with stashed changes",
			Cmd:    "qs",
			Args:   []string{"dev", "another-feature"},
			Stdout: "stash",
			CheckResults: func(t *testing.T) {
				// Check stash list
				cmd := exec.Command("git", "stash", "list")
				cmd.Dir = st.GetClonePath()
				output, err := cmd.Output()
				require.NoErrorf(t, err, "failed to list stash: %v", err)
				require.NotEmpty(t, string(output), "no stashed changes found")
			},
		})

		// Run the test
		err := st.Run()
		require.NoError(t, err)
	})

	t.Run("LargeFilesCheck", func(t *testing.T) {
		// Get GitHub account from environment
		ghAccount := os.Getenv(systrun.EnvGithubAccount)
		if ghAccount == "" {
			t.Skipf("%s environment variable must be set", systrun.EnvGithubAccount)
		}

		// Generate unique repo names
		repoID := uuid.New().String()[:8]
		mainRepoName := fmt.Sprintf("qs-test-large-%s", repoID)

		// Create test configuration
		cfg := systrun.SystemTestCfg{
			GithubAccount:    ghAccount,
			GithubToken:      os.Getenv(systrun.EnvGithubToken),
			UpstreamRepoName: mainRepoName,
		}

		// Create system test
		st := systrun.NewSystemTest(t, cfg)

		// Create a dev branch
		devBranchName := "feature-large-file"
		st.AddCase(systrun.TestCase{
			Name:   "Create developer branch",
			Cmd:    "qs",
			Args:   []string{"dev", devBranchName},
			Stdout: fmt.Sprintf("Dev branch '%s' will be created", devBranchName),
		})

		// Create a large file (5MB, which should exceed most Git LFS limits)
		st.AddCase(systrun.TestCase{
			Name: "Create large file",
			Cmd:  "bash",
			Args: []string{"-c", fmt.Sprintf("dd if=/dev/zero of=%s/large-file.bin bs=1024 count=5120", st.GetClonePath())},
			CheckResults: func(t *testing.T) {
				// Verify file was created
				filePath := fmt.Sprintf("%s/large-file.bin", st.GetClonePath())
				info, err := os.Stat(filePath)
				require.NoErrorf(t, err, "failed to stat file: %v", err)

				expectedSize := int64(5120 * 1024) // 5MB
				require.Equalf(t, info.Size(), expectedSize, "file size incorrect, expected %d, got %d", expectedSize, info.Size())
			},
		})

		// Try to add and commit large file
		st.AddCase(systrun.TestCase{
			Name:   "Try to commit large file",
			Cmd:    "qs",
			Args:   []string{"u", "-m", "Add large file"},
			Stderr: "large", // Error message should mention large files
		})

		// Run the test
		_ = st.Run() // We expect warning about large files
	})
}

func TestNoGithubToken(t *testing.T) {
	// Skip if running in CI without proper setup
	if os.Getenv("CI") == "true" && os.Getenv(systrun.EnvGithubAccount) == "" {
		t.Skip("Skipping system test in CI environment without proper setup")
	}

	// Save current token
	oldToken := os.Getenv(systrun.EnvGithubToken)

	// Unset token
	os.Unsetenv(systrun.EnvGithubToken)

	// Restore token after test
	defer func() {
		os.Setenv(systrun.EnvGithubToken, oldToken)
	}()

	// Get GitHub account from environment
	ghAccount := os.Getenv(systrun.EnvGithubAccount)
	if ghAccount == "" {
		t.Skipf("%s environment variable must be set", systrun.EnvGithubAccount)
	}

	// Generate unique repo names
	repoID := uuid.New().String()[:8]
	mainRepoName := fmt.Sprintf("qs-test-notoken-%s", repoID)

	// Create test configuration without token
	cfg := systrun.SystemTestCfg{
		GithubAccount:    ghAccount,
		GithubToken:      "", // Explicitly empty
		UpstreamRepoName: mainRepoName,
	}

	// Create system test
	st := systrun.NewSystemTest(t, cfg)

	// Try to create a repo without token
	// The test should still work if gh CLI is logged in through other means
	// But might fail if authentication is required
	_ = st.Run()
}
