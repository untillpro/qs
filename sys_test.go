package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/atotto/clipboard"
	"github.com/stretchr/testify/require"
	"github.com/untillpro/qs/internal/systrun"
)

// TestForkOnExistingFork tests the case where a fork already exists
func TestFork_OnExistingFork(t *testing.T) {
	t.Skip()
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "fork",
		},
		UpstreamState:  systrun.RemoteStateOK,
		ForkState:      systrun.RemoteStateOK,
		ExpectedStderr: "you are in fork already",
		Expectations:   []systrun.ExpectationFunc{systrun.ExpectationForkExists},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.Error(err)

}

// TestFork tests the case where a fork does not exist yet
func TestFork(t *testing.T) {
	t.Skip()
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "fork",
		},
		UpstreamState: systrun.RemoteStateOK,
		ForkState:     systrun.RemoteStateNull,
		Expectations:  []systrun.ExpectationFunc{systrun.ExpectationForkExists},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestForkNoRemotes tests the case where there is no origin remote
func TestFork_NoRemotes(t *testing.T) {
	t.Skip()
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "fork",
		},
		UpstreamState:  systrun.RemoteStateNull,
		ForkState:      systrun.RemoteStateNull,
		ExpectedStderr: "origin remote not found",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.Error(err)
}

// TestDevCustomName tests creating a new dev branch when it doesn't exist
func TestDev_CustomName(t *testing.T) {
	t.Skip()
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "dev",
			Stdin:   "y",
		},
		ClipboardContent: systrun.ClipboardContentCustom,
		UpstreamState:    systrun.RemoteStateOK,
		ForkState:        systrun.RemoteStateOK,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationCustomBranchIsCurrentBranch,
			systrun.ExpectationLargeFileHooksInstalled,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestDev_NoUpstream_CustomName tests creating a new dev branch when it doesn't exist
func TestDev_NoUpstream_CustomName(t *testing.T) {
	t.Skip()
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"--no-fork"},
			Stdin:   "y",
		},
		ClipboardContent: systrun.ClipboardContentCustom,
		UpstreamState:    systrun.RemoteStateOK,
		ForkState:        systrun.RemoteStateNull,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationCustomBranchIsCurrentBranch,
			systrun.ExpectationLargeFileHooksInstalled,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestDevExistingBranch tests behavior when dev branch already exists
func TestDev_ExistingBranch(t *testing.T) {
	require := require.New(t)

	err := clipboard.WriteAll("")
	require.NoError(err)

	branchName := "dev"
	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "dev",
			Args:    []string{branchName},
			Stdin:   "y",
		},
		UpstreamState:  systrun.RemoteStateOK,
		ForkState:      systrun.RemoteStateOK,
		DevBranchState: systrun.DevBranchStateExistsButNotCheckedOut,
		ExpectedStderr: fmt.Sprintf("dev branch '%s' already exists", branchName),
	}

	sysTest := systrun.New(t, testConfig)
	err = sysTest.Run()
	require.Error(err)
}

// TestDevNoForkExistingIssue tests creating a dev branch when upstream remote doesn't exist
func TestDev_NoFork_ExistingIssue(t *testing.T) {
	t.Skip()
	require := require.New(t)

	ghConfig := getGithubConfig(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: ghConfig,
		CommandConfig: systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"--no-fork"},
			Stdin:   "y",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateNull,
		ClipboardContent:  systrun.ClipboardContentGithubIssue,
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationBranchLinkedToIssue,
			systrun.ExpectationLargeFileHooksInstalled,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestPR_FromOtherClone(t *testing.T) {
	t.Skip()
	require := require.New(t)

	ghConfig := getGithubConfig(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: ghConfig,
		CommandConfig: systrun.CommandConfig{
			Command: "pr",
		},
		UpstreamState:              systrun.RemoteStateOK,
		ForkState:                  systrun.RemoteStateOK,
		ClipboardContent:           systrun.ClipboardContentGithubIssue,
		SyncState:                  systrun.SyncStateSynchronized,
		RunCommandFromAnotherClone: true,
		NeedCollaboration:          true,
		Expectations:               []systrun.ExpectationFunc{systrun.ExpectationPRCreated},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestDev_NoFork_NonExistingIssue tests creating a dev branch when upstream remote doesn't exist
func TestDev_NoFork_NonExistingIssue(t *testing.T) {
	t.Skip()
	require := require.New(t)

	ghConfig := getGithubConfig(t)
	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: ghConfig,
		CommandConfig: systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"--no-fork"},
			Stdin:   "y",
		},
		ClipboardContent: systrun.ClipboardContentUnavailableGithubIssue,
		UpstreamState:    systrun.RemoteStateOK,
		ForkState:        systrun.RemoteStateNull,
		ExpectedStderr:   "Invalid GitHub issue link",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.Error(err)
}

// TestDevNoForkJiraTicketURL tests creating a dev branch with a valid JIRA ticket URL
func TestDev_NoFork_JiraTicketURL(t *testing.T) {
	t.Skip()
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"--no-fork"},
			Stdin:   "y",
		},
		UpstreamState:    systrun.RemoteStateOK,
		ForkState:        systrun.RemoteStateNull,
		ClipboardContent: systrun.ClipboardContentJiraTicket,
		Expectations:     []systrun.ExpectationFunc{systrun.ExpectationCurrentBranchHasPrefix},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestPR tests creating a basic PR
func TestPR_Synchronized(t *testing.T) {
	t.Skip()
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "pr",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateOK,
		SyncState:         systrun.SyncStateSynchronized,
		ClipboardContent:  systrun.ClipboardContentGithubIssue,
		NeedCollaboration: true,
		Expectations:      []systrun.ExpectationFunc{systrun.ExpectationPRCreated},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestPR_ForkChanged(t *testing.T) {
	t.Skip()
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "pr",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateOK,
		SyncState:         systrun.SyncStateForkChanged,
		ClipboardContent:  systrun.ClipboardContentGithubIssue,
		NeedCollaboration: true,
		Expectations:      []systrun.ExpectationFunc{systrun.ExpectationPRCreated},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.Error(err)
}

// TestDownload tests synchronizing local repository with remote changes
func TestDownload(t *testing.T) {
	t.Skip()
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "d",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateOK,
		SyncState:         systrun.SyncStateForkChanged,
		ClipboardContent:  systrun.ClipboardContentGithubIssue,
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationCloneIsSyncedWithFork,
			systrun.ExpectationNotesDownloaded,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestUpload tests uploading local changes to remote repository
func TestUpload(t *testing.T) {
	t.Skip()
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "u",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateOK,
		SyncState:         systrun.SyncStateUncommitedChangesInClone,
		ClipboardContent:  systrun.ClipboardContentGithubIssue,
		NeedCollaboration: true,
		Expectations:      []systrun.ExpectationFunc{systrun.ExpectationRemoteBranchWithCommitMessage},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
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
