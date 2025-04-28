package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/qs/internal/systrun"
)

// TestForkNonExistingFork tests the case where a fork does not exist yet
func TestForkNonExistingFork(t *testing.T) {
	require := require.New(t)

	ghConfig := getGithubConfig(t)

	testConfig := &systrun.TestConfig{
		TestID:         "fork-non-existing",
		GHConfig:       getGithubConfig(t),
		Command:        "fork",
		Args:           []string{systrun.GithubURL + "/" + ghConfig.UpstreamAccount + "/test-repo"},
		UpstreamState:  systrun.RemoteStateOK,
		ForkState:      systrun.RemoteStateNull,
		ExpectedStdout: "Repository forked successfully",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestForkExistingFork tests the case where a fork already exists
func TestForkExistingFork(t *testing.T) {
	require := require.New(t)

	ghConfig := getGithubConfig(t)

	testConfig := &systrun.TestConfig{
		TestID:         "fork-existing",
		GHConfig:       getGithubConfig(t),
		Command:        "fork",
		Args:           []string{systrun.GithubURL + "/" + ghConfig.UpstreamAccount + "/test-repo"},
		UpstreamState:  systrun.RemoteStateOK,
		ForkState:      systrun.RemoteStateOK,
		ExpectedStdout: "Fork already exists",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestForkNoOriginRemote tests the case where there is no origin remote
func TestForkNoOriginRemote(t *testing.T) {
	require := require.New(t)

	ghConfig := getGithubConfig(t)
	testConfig := &systrun.TestConfig{
		TestID:         "fork-no-origin",
		GHConfig:       getGithubConfig(t),
		Command:        "fork",
		Args:           []string{systrun.GithubURL + "/" + ghConfig.UpstreamAccount + "/test-repo"},
		UpstreamState:  systrun.RemoteStateNull,
		ForkState:      systrun.RemoteStateNull,
		ExpectedStderr: "origin remote not found",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.Error(err)
}

// TestDevNewBranch tests creating a new dev branch when it doesn't exist
func TestDevNewBranch(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:          "dev-new-branch",
		GHConfig:        getGithubConfig(t),
		Command:         "dev",
		Args:            []string{},
		UpstreamState:   systrun.RemoteStateOK,
		ForkState:       systrun.RemoteStateOK,
		DevBranchExists: false,
		ExpectedStdout:  "New dev branch created",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestDevExistingBranch tests behavior when dev branch already exists
func TestDevExistingBranch(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:          "dev-existing-branch",
		GHConfig:        getGithubConfig(t),
		Command:         "dev",
		Args:            []string{},
		UpstreamState:   systrun.RemoteStateOK,
		ForkState:       systrun.RemoteStateOK,
		DevBranchExists: true,
		ExpectedStdout:  "Dev branch already exists",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestDevNoFork tests creating a dev branch when fork doesn't exist
func TestDevNoFork(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:          "dev-no-fork",
		GHConfig:        getGithubConfig(t),
		Command:         "dev",
		Args:            []string{},
		UpstreamState:   systrun.RemoteStateOK,
		ForkState:       systrun.RemoteStateNull,
		DevBranchExists: false,
		ExpectedStdout:  "Creating dev branch in upstream repo",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestPRBasic tests creating a basic PR
func TestPRBasic(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:         "pr-basic",
		GHConfig:       getGithubConfig(t),
		Command:        "pr",
		Args:           []string{},
		UpstreamState:  systrun.RemoteStateOK,
		ForkState:      systrun.RemoteStateOK,
		SyncState:      systrun.SyncStateSynchronized,
		ExpectedStdout: "Creating pull request",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestPRDevBranchOutOfDate tests behavior when dev branch is out of date
func TestPRDevBranchOutOfDate(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:         "pr-branch-out-of-date",
		GHConfig:       getGithubConfig(t),
		Command:        "pr",
		Args:           []string{},
		UpstreamState:  systrun.RemoteStateOK,
		ForkState:      systrun.RemoteStateOK,
		SyncState:      systrun.SyncStateForkChanged,
		ExpectedStderr: "This branch is out-of-date. Merge automatically [y/n]?",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.Error(err)
}

// TestPRWrongBranch tests PR creation when not on dev branch
func TestPRWrongBranch(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:         "pr-wrong-branch",
		GHConfig:       getGithubConfig(t),
		Command:        "pr",
		Args:           []string{},
		UpstreamState:  systrun.RemoteStateOK,
		ForkState:      systrun.RemoteStateOK,
		SyncState:      systrun.SyncStateDoesntTrackOrigin,
		ExpectedStderr: "You are not on dev branch",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.Error(err)
}

// TestDownload tests synchronizing local repository with remote changes
func TestDownload(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:         "download",
		GHConfig:       getGithubConfig(t),
		Command:        "d",
		Args:           []string{},
		UpstreamState:  systrun.RemoteStateOK,
		ForkState:      systrun.RemoteStateOK,
		SyncState:      systrun.SyncStateForkChanged,
		ExpectedStdout: "Downloading changes from remote",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestUpload tests uploading local changes to remote repository
func TestUpload(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:         "upload",
		GHConfig:       getGithubConfig(t),
		Command:        "u",
		Args:           []string{},
		UpstreamState:  systrun.RemoteStateOK,
		ForkState:      systrun.RemoteStateOK,
		SyncState:      systrun.SyncStateCloneChanged,
		ExpectedStdout: "Uploading changes to remote",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestUploadConflict tests uploading changes when there are conflicts
func TestUploadConflict(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:         "upload-conflict",
		GHConfig:       getGithubConfig(t),
		Command:        "u",
		Args:           []string{},
		UpstreamState:  systrun.RemoteStateOK,
		ForkState:      systrun.RemoteStateOK,
		SyncState:      systrun.SyncStateBothChangedConflict,
		ExpectedStderr: "There are conflicts that need to be resolved manually",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.Error(err)
}

// getGithubConfig retrieves GitHub credentials from environment variables
// and skips the test if any credentials are missing
func getGithubConfig(t *testing.T) systrun.GithubConfig {
	upstreamAccount := os.Getenv(systrun.EnvUpstreamGithubAccount)
	upstreamToken := os.Getenv(systrun.EnvUpstreamGithubToken)
	forkAccount := os.Getenv(systrun.EnvForkGithubAccount)
	forkToken := os.Getenv(systrun.EnvForkGithubToken)

	// Skip test if credentials are not set
	if upstreamAccount == "" || upstreamToken == "" || forkAccount == "" || forkToken == "" {
		t.Skip("GitHub credentials not set, skipping test")
	}

	return systrun.GithubConfig{
		UpstreamAccount: upstreamAccount,
		UpstreamToken:   upstreamToken,
		ForkAccount:     forkAccount,
		ForkToken:       forkToken,
	}
}
