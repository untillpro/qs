package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/atotto/clipboard"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/untillpro/qs/internal/commands"
	"github.com/untillpro/qs/internal/systrun"
)

// TestForkOnExistingFork tests the case where a fork already exists
func TestForkOnExistingFork(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   "fork-existing",
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "fork",
		},
		UpstreamState:  systrun.RemoteStateOK,
		ForkState:      systrun.RemoteStateOK,
		ExpectedStdout: "you are in fork already",
		Expectations: []systrun.IExpectation{
			systrun.ExpectedForkExists{},
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)

}

// TestFork tests the case where a fork does not exist yet
func TestFork(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   "fork",
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "fork",
		},
		UpstreamState: systrun.RemoteStateOK,
		ForkState:     systrun.RemoteStateNull,
		Expectations: []systrun.IExpectation{
			systrun.ExpectedRemoteState{
				UpstreamRemoteState: systrun.RemoteStateOK,
				ForkRemoteState:     systrun.RemoteStateOK,
			},
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestForkNoOriginRemote tests the case where there is no origin remote
func TestForkNoOriginRemote(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   "fork-no-origin",
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
func TestDevCustomName(t *testing.T) {
	require := require.New(t)

	branchName := "branch-name"
	testConfig := &systrun.TestConfig{
		TestID:   "dev-custom-name",
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "dev",
			Args:    []string{},
			Stdin:   "y",
		},
		ClipboardContent: branchName,
		UpstreamState:    systrun.RemoteStateOK,
		ForkState:        systrun.RemoteStateOK,
		Expectations: []systrun.IExpectation{
			systrun.ExpectedDevBranch{
				BranchName: branchName,
			},
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestDevCustomName tests creating a new dev branch when it doesn't exist
func TestDevNoForkCustomName(t *testing.T) {
	require := require.New(t)

	branchName := "branch-name"
	testConfig := &systrun.TestConfig{
		TestID:   "dev-custom-name",
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"--no-fork"},
			Stdin:   "y",
		},
		ClipboardContent: branchName,
		UpstreamState:    systrun.RemoteStateOK,
		ForkState:        systrun.RemoteStateOK,
		Expectations: []systrun.IExpectation{
			systrun.ExpectedDevBranch{
				BranchName: branchName,
			},
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestDevExistingBranch tests behavior when dev branch already exists
func TestDevExistingBranch(t *testing.T) {
	require := require.New(t)

	err := clipboard.WriteAll("")
	require.NoError(err)

	branchName := "dev"
	testConfig := &systrun.TestConfig{
		TestID:   "dev-existing-branch",
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "dev",
			Args:    []string{branchName},
			Stdin:   "y",
		},
		UpstreamState:  systrun.RemoteStateOK,
		ForkState:      systrun.RemoteStateOK,
		Branches:       []string{branchName},
		ExpectedStdout: fmt.Sprintf("dev branch '%s' already exists", branchName),
	}

	sysTest := systrun.New(t, testConfig)
	err = sysTest.Run()
	require.NoError(err)
}

// TODO: Add same test for case when both exist upstream and fork remotes
// TestDevNoForkExistingIssue tests creating a dev branch when upstream remote doesn't exist
func TestDevNoForkExistingIssue(t *testing.T) {
	require := require.New(t)

	ghConfig := getGithubConfig(t)

	testConfig := &systrun.TestConfig{
		TestID:   "dev-no-fork-existing-issue",
		GHConfig: ghConfig,
		CommandConfig: systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"--no-fork"},
			Stdin:   "y",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateNull,
		UseNewGithubIssue: true,
		Expectations: []systrun.IExpectation{
			systrun.ExpectedBranchLinkedToIssue{
				// Get issue URL from RuntimeEnvironment
				// IssueID: "1",
			},
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TODO: Add same test for case when both exist upstream and fork remotes
// TestDevNoForkExistingIssue tests creating a dev branch when upstream remote doesn't exist
func TestDevNoForkNonExistingIssue(t *testing.T) {
	require := require.New(t)

	ghConfig := getGithubConfig(t)

	nonExistingIssueURL := fmt.Sprintf("https://github.com/%s/%s/issues/abc", ghConfig.UpstreamAccount, uuid.New().String())
	testConfig := &systrun.TestConfig{
		TestID:   "dev-no-fork-non-existing-issue",
		GHConfig: ghConfig,
		CommandConfig: systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"--no-fork"},
			Stdin:   "y",
		},
		ClipboardContent: nonExistingIssueURL,
		UpstreamState:    systrun.RemoteStateOK,
		ForkState:        systrun.RemoteStateNull,
		ExpectedStderr:   "Issue not found",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TODO: Add same test for case when both exist upstream and fork remotes
// TestDevNoForkForeignIssue tests creating a dev branch when upstream remote doesn't exist
func TestDevNoForkForeignIssue(t *testing.T) {
	require := require.New(t)

	ghConfig := getGithubConfig(t)

	testConfig := &systrun.TestConfig{
		TestID:   "dev-no-fork-foreign-issue",
		GHConfig: ghConfig,
		CommandConfig: systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"--no-fork"},
			Stdin:   "y",
		},
		UpstreamState:                    systrun.RemoteStateOK,
		ForkState:                        systrun.RemoteStateNull,
		UseNewGithubIssueFromForeignRepo: true,
		Expectations: []systrun.IExpectation{
			systrun.ExpectedBranchLinkedToIssue{},
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestDevNoForkJiraTicketURL tests creating a dev branch with a valid JIRA ticket URL
func TestDevNoForkJiraTicketURL(t *testing.T) {
	require := require.New(t)

	jiraTicketURL := os.Getenv("JIRA_TICKET_URL")
	if jiraTicketURL == "" {
		t.Skip("JIRA_TICKET_URL environment variable not set, skipping test")
	}

	jiraTicketID, ok := commands.GetJiraTicketIDFromArgs(jiraTicketURL)
	if !ok {
		t.Fatalf("Invalid JIRA ticket URL: %s", jiraTicketURL)
	}

	testConfig := &systrun.TestConfig{
		TestID:   "dev-no-fork-jira-ticket-url",
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"--no-fork", jiraTicketURL},
			Stdin:   "y",
		},
		UpstreamState: systrun.RemoteStateOK,
		ForkState:     systrun.RemoteStateNull,
		Expectations: []systrun.IExpectation{
			systrun.ExpectedCurrentBranchHasPrefix{
				Prefix: jiraTicketID,
			},
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestPRBasic tests creating a basic PR
func TestPRBasic(t *testing.T) {
	t.Skip("Test is under debugging session")

	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   "pr-basic",
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "pr",
			Args:    []string{},
		},
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
	t.Skip("Test is under debugging session")

	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   "pr-branch-out-of-date",
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "pr",
			Args:    []string{},
		},
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
	t.Skip("Test is under debugging session")

	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   "pr-wrong-branch",
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "pr",
			Args:    []string{},
		},
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
	t.Skip("Test is under debugging session")

	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   "download",
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "d",
			Args:    []string{},
		},
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
	t.Skip("Test is under debugging session")

	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   "upload",
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "u",
			Args:    []string{},
		},
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
	t.Skip("Test is under debugging session")

	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   "upload-conflict",
		GHConfig: getGithubConfig(t),
		CommandConfig: systrun.CommandConfig{
			Command: "u",
			Args:    []string{},
		},
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
