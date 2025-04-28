package systrun

import (
	"testing"
)

// SystemTest represents a single system test for the qs utility
type SystemTest struct {
	t             *testing.T
	cfg           *TestConfig
	cloneRepoPath string
}

// TestConfig contains all configuration for a system test
type TestConfig struct {
	TestID         string
	GHConfig       GithubConfig
	Command        string
	Args           []string
	ExpectedStderr string
	ExpectedStdout string
	UpstreamState  RemoteState
	ForkState      RemoteState
	SyncState      SyncState
	// DevBranchExists indicates if the dev branch exists in the clone repo
	DevBranchExists bool
}

// GithubConfig holds GitHub account and token information
type GithubConfig struct {
	UpstreamAccount string
	UpstreamToken   string
	ForkAccount     string
	ForkToken       string
}
