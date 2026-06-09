package commands

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkGitLsRemote(b *testing.B) {
	wd := repoRoot(b)
	b.ResetTimer()
	for b.Loop() {
		cmd := exec.Command("git", "ls-remote", "--heads", "origin", "main")
		cmd.Dir = wd
		out, err := cmd.Output()
		require.NoError(b, err)
		_ = out
	}
}

// repoRoot returns the root of the current git repository.
func repoRoot(b *testing.B) string {
	b.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(b, err)
	return string(out[:len(out)-1]) // trim trailing newline
}

