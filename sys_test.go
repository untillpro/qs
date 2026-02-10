package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/qs/gitcmds"
	"github.com/untillpro/qs/internal/systrun"
)

// TestForkOnExistingFork tests the case where a fork already exists
func TestFork_OnExistingFork(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
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
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
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
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
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
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
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

// TestDevExistingBranch tests behavior when dev branch already exists
func TestDev_ExistingBranch(t *testing.T) {
	require := require.New(t)

	branchName := "branch-name"
	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Args:    []string{branchName},
			Stdin:   "y",
		},
		UpstreamState: systrun.RemoteStateOK,
		ForkState:     systrun.RemoteStateOK,
		BranchState: &systrun.BranchState{
			DevBranchExists: true,
		},
		ExpectedStderr: fmt.Sprintf("dev branch %s-dev already exists", branchName),
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()

	require.Error(err)
}

// TestDev_ExistingBranch_NoUpstream tests behavior when dev branch already exists
func TestDev_ExistingBranch_NoUpstream(t *testing.T) {
	require := require.New(t)

	branchName := "branch-name"
	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Args:    []string{branchName},
			Stdin:   "y",
		},
		UpstreamState: systrun.RemoteStateOK,
		BranchState: &systrun.BranchState{
			DevBranchExists: true,
		},
		ExpectedStderr: fmt.Sprintf("dev branch %s-dev already exists", branchName),
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()

	require.Error(err)
}

func TestDev_GitHubIssue_NoUpstream(t *testing.T) {
	require := require.New(t)

	ghConfig := getGithubConfig(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: ghConfig,
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
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
	require := require.New(t)

	ghConfig := getGithubConfig(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: ghConfig,
		CommandConfig: &systrun.CommandConfig{
			Command: "pr",
		},
		UpstreamState:          systrun.RemoteStateOK,
		ForkState:              systrun.RemoteStateOK,
		ClipboardContent:       systrun.ClipboardContentGithubIssue,
		SyncState:              systrun.SyncStateSynchronized,
		RunCommandOnOtherClone: true,
		NeedCollaboration:      true,
		Expectations:           []systrun.ExpectationFunc{systrun.ExpectationPRCreated},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestPR_FromOtherClone_NoUpstream(t *testing.T) {
	require := require.New(t)

	ghConfig := getGithubConfig(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: ghConfig,
		CommandConfig: &systrun.CommandConfig{
			Command: "pr",
		},
		UpstreamState:          systrun.RemoteStateOK,
		ClipboardContent:       systrun.ClipboardContentGithubIssue,
		SyncState:              systrun.SyncStateSynchronized,
		RunCommandOnOtherClone: true,
		NeedCollaboration:      true,
		Expectations:           []systrun.ExpectationFunc{systrun.ExpectationPRCreated},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestDev_NonExistingIssue_NoUpstream tests creating a dev branch when upstream remote doesn't exist
// Auto-detection handles single remote mode (no --no-fork flag needed)
func TestDev_NonExistingIssue_NoUpstream(t *testing.T) {
	require := require.New(t)

	ghConfig := getGithubConfig(t)
	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: ghConfig,
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Stdin:   "y",
		},
		ClipboardContent: systrun.ClipboardContentUnavailableGithubIssue,
		UpstreamState:    systrun.RemoteStateOK,
		ExpectedStderr:   "invalid GitHub issue link",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.Error(err)
}

func TestDev_JiraTicketURL_NoUpstream(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Stdin:   "y",
		},
		UpstreamState:    systrun.RemoteStateOK,
		ClipboardContent: systrun.ClipboardContentJiraTicket,
		Expectations:     []systrun.ExpectationFunc{systrun.ExpectationCurrentBranchHasPrefix},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestDev_JiraTicketURL_InvalidJiraAPIToken(t *testing.T) {
	require := require.New(t)

	// Set JIRA related environment variables
	apiToken := os.Getenv("JIRA_API_TOKEN")
	err := os.Setenv("JIRA_API_TOKEN", "invalid_token")
	require.NoError(err)
	defer func() {
		_ = os.Setenv("JIRA_API_TOKEN", apiToken)
	}()

	jiraEmail := os.Getenv("JIRA_EMAIL")
	err = os.Setenv("JIRA_EMAIL", "user@server.com")
	require.NoError(err)
	defer func() {
		_ = os.Setenv("JIRA_EMAIL", jiraEmail)
	}()

	jiraTicketURL := os.Getenv("JIRA_TICKET_URL")
	err = os.Setenv("JIRA_TICKET_URL", "https://untill.atlassian.net/browse/AIR-2814")
	require.NoError(err)
	defer func() {
		_ = os.Setenv("JIRA_TICKET_URL", jiraTicketURL)
	}()

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Stdin:   "y",
		},
		UpstreamState:    systrun.RemoteStateOK,
		ClipboardContent: systrun.ClipboardContentJiraTicket,
		ExpectedStdout: []string{
			"Issue does not exist or you do not have permission to see it",
			"Your JIRA_API_TOKEN environment variable is set and valid",
			"The Jira ticket doesn't exist",
			"You don't have permission to view it",
		},
	}

	sysTest := systrun.New(t, testConfig)
	err = sysTest.Run()
	require.NoError(err)
}

// TestPR_Synchronized tests creating a basic PR
func TestPR_Synchronized(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
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

// TestPR_Synchronized_NoUpstream tests creating a basic PR
func TestPR_Synchronized_NoUpstream(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "pr",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateNull,
		SyncState:         systrun.SyncStateSynchronized,
		ClipboardContent:  systrun.ClipboardContentGithubIssue,
		NeedCollaboration: true,
		Expectations:      []systrun.ExpectationFunc{systrun.ExpectationPRCreated},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestPR_FromJiraTicket tests creating a PR on Jira based branch
func TestPR_FromJiraTicket(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "pr",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateOK,
		SyncState:         systrun.SyncStateSynchronized,
		ClipboardContent:  systrun.ClipboardContentJiraTicket,
		NeedCollaboration: true,
		Expectations:      []systrun.ExpectationFunc{systrun.ExpectationPRCreated},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestPR_ForkChanged(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
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

func TestPR_UpstreamChanged_NoFork(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "pr",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateNull,
		SyncState:         systrun.SyncStateForkChanged,
		ClipboardContent:  systrun.ClipboardContentGithubIssue,
		NeedCollaboration: true,
		Expectations:      []systrun.ExpectationFunc{systrun.ExpectationPRCreated},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.Error(err)
}

func TestPR_NoUpstream(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "pr",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateNull,
		SyncState:         systrun.SyncStateCloneChanged,
		ClipboardContent:  systrun.ClipboardContentGithubIssue,
		NeedCollaboration: true,
		Expectations:      []systrun.ExpectationFunc{systrun.ExpectationPRCreated},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestDownload tests synchronizing local repository with remote changes
func TestDownload(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
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
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "u",
			Stdin:   "y",
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

// TestUpload_NothingToCommit tests pushing when there are no uncommitted changes but unpushed commits exist
func TestUpload_NothingToCommit(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "u",
			Stdin:   "y",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateOK,
		SyncState:         systrun.SyncStateCloneIsAheadOfFork,
		ClipboardContent:  systrun.ClipboardContentGithubIssue,
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationCloneIsSyncedWithFork,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestDevD_DevBranch_NoRT_NoPR - develop branch exists, no remote tracking branch, no pull request
func TestDevD_DevBranch_NoRT_NoPR(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"-d"},
			Stdin:   "y",
		},
		UpstreamState: systrun.RemoteStateOK,
		BranchState: &systrun.BranchState{
			DevBranchExists: true,
		},
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationTwoLocalBranches,
			systrun.ExpectationOneRemoteBranch,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestDevD_DevBranch_RT_NoPR(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"-d"},
			Stdin:   "y",
		},
		UpstreamState: systrun.RemoteStateOK,
		BranchState: &systrun.BranchState{
			DevBranchExists:      true,
			DevBranchHasRtBranch: true,
		},
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationTwoLocalBranches,
			systrun.ExpectationTwoRemoteBranches,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestDevD_DevBranch_RT_PROpen(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"-d"},
			Stdin:   "y",
		},
		UpstreamState: systrun.RemoteStateOK,
		ForkState:     systrun.RemoteStateOK,
		BranchState: &systrun.BranchState{
			DevBranchExists:      true,
			DevBranchHasRtBranch: true,
			PRExists:             true,
		},
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationTwoLocalBranches,
			systrun.ExpectationTwoRemoteBranches,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestDevD_DevBranch_RT_PRMerged(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"-d"},
			Stdin:   "y",
		},
		UpstreamState: systrun.RemoteStateOK,
		ForkState:     systrun.RemoteStateOK,
		BranchState: &systrun.BranchState{
			DevBranchExists:      true,
			DevBranchHasRtBranch: true,
			PRMerged:             true,
		},
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationOneLocalBranch,
			systrun.ExpectationOneRemoteBranch,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestDevD_DevBranch_NoRT_PRMerged(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"-d"},
			Stdin:   "y",
		},
		UpstreamState: systrun.RemoteStateOK,
		ForkState:     systrun.RemoteStateOK,
		BranchState: &systrun.BranchState{
			DevBranchExists: true,
			PRMerged:        true,
		},
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationOneLocalBranch,
			systrun.ExpectationOneRemoteBranch,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestDevD_PrBranch_NoRT_PRMerged(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"-d"},
			Stdin:   "y",
		},
		UpstreamState: systrun.RemoteStateOK,
		ForkState:     systrun.RemoteStateOK,
		BranchState: &systrun.BranchState{
			PRBranchExists: true,
			PRMerged:       true,
		},
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationOneLocalBranch,
			systrun.ExpectationOneRemoteBranch,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestDevD_PrBranch_RT_PRMerged(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"-d"},
			Stdin:   "y",
		},
		UpstreamState: systrun.RemoteStateOK,
		ForkState:     systrun.RemoteStateOK,
		BranchState: &systrun.BranchState{
			PRBranchExists:      true,
			PRBranchHasRtBranch: true,
			PRMerged:            true,
		},
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationOneLocalBranch,
			systrun.ExpectationOneRemoteBranch,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestDevD_PrBranch_NoRT_PROpen(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"-d"},
			Stdin:   "y",
		},
		UpstreamState: systrun.RemoteStateOK,
		ForkState:     systrun.RemoteStateOK,
		BranchState: &systrun.BranchState{
			PRBranchExists: true,
			PRExists:       true,
		},
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationTwoLocalBranches,
			systrun.ExpectationOneRemoteBranch,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestDevD_DevBranch_PrBranch_PROpen(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"-d"},
			Stdin:   "y",
		},
		UpstreamState: systrun.RemoteStateOK,
		ForkState:     systrun.RemoteStateOK,
		BranchState: &systrun.BranchState{
			DevBranchExists: true,
			PRBranchExists:  true,
			PRExists:        true,
		},
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationThreeLocalBranches,
			systrun.ExpectationOneRemoteBranch,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestDevD_DevBranch_PrBranch_PRMerged(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"-d"},
			Stdin:   "y",
		},
		UpstreamState: systrun.RemoteStateOK,
		ForkState:     systrun.RemoteStateOK,
		BranchState: &systrun.BranchState{
			DevBranchExists: true,
			PRBranchExists:  true,
			PRMerged:        true,
		},
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationOneLocalBranch,
			systrun.ExpectationOneRemoteBranch,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestDevD_NoBranches(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"-d"},
			Stdin:   "y",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateOK,
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationOneLocalBranch,
			systrun.ExpectationOneRemoteBranch,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

func TestQS(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:           strings.ToLower(t.Name()),
		GHConfig:         getGithubConfig(t),
		CommandConfig:    &systrun.CommandConfig{},
		UpstreamState:    systrun.RemoteStateOK,
		ForkState:        systrun.RemoteStateOK,
		SyncState:        systrun.SyncStateUncommitedChangesInClone,
		ClipboardContent: systrun.ClipboardContentGithubIssue,
		ExpectedStdout: []string{
			"## ",
			"origin\t",
			"upstream\t",
			"Summary:",
			"Total positive delta: ",
			"Largest positive delta: ",
			"0/1.txt",
			"0/1/2.txt",
			"0/1/2/3.txt",
		},
		NeedCollaboration: true,
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestLargeFileHook tests that the large file precommit hook is functional
func TestLargeFileHook(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Stdin:   "y",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateNull,
		ClipboardContent:  systrun.ClipboardContentGithubIssue,
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationBranchLinkedToIssue,
			systrun.ExpectationLargeFileHooksInstalled,
			systrun.ExpectationLargeFileHookFunctional,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestPR_FromJiraTicket_NoUpstream tests creating a PR in single remote mode
// with automatic workflow detection
func TestPR_FromJiraTicket_NoUpstream(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "pr",
		},
		ClipboardContent:  systrun.ClipboardContentJiraTicket,
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateNull,
		SyncState:         systrun.SyncStateSynchronized,
		NeedCollaboration: true,
		Expectations:      []systrun.ExpectationFunc{systrun.ExpectationPRCreated},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestDev_CustomName_NoUpstream tests creating a dev branch in single remote mode
// with automatic workflow detection
func TestDev_CustomName_NoUpstream(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Stdin:   "y",
		},
		ClipboardContent: systrun.ClipboardContentCustom,
		UpstreamState:    systrun.RemoteStateOK,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationCustomBranchIsCurrentBranch,
			systrun.ExpectationLargeFileHooksInstalled,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestDev_MainBranchConflict tests AIR-1959: handling non-mergeable main branch properly
// This test verifies that when there's a conflict in the main branch (local main diverged
// from upstream/main and cannot be rebased), the tool provides a helpful workaround message
func TestDev_MainBranchConflict(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Stdin:   "y",
		},
		ClipboardContent:  systrun.ClipboardContentGithubIssue,
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateOK,
		SyncState:         systrun.SyncStateMainBranchConflict,
		NeedCollaboration: true,
		ExpectedStdout: []string{
			"A conflict is detected in main branch", // Part of MsgConflictDetected
			gitcmds.MsgGitCheckoutMain,
			gitcmds.MsgGitFetchUpstream,
			gitcmds.MsgGitResetHardUpstream,
			gitcmds.MsgGitPushOriginMainForce,
			"Warning: This will overwrite your main branch", // Part of MsgWarningOverwriteMainBranch
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	// The command should fail with an error because of the conflict
	require.Error(err)
}

// TestDevD_MainBranchDiverged tests AIR-2783: fast-forward only merge failure
// This test verifies that when fork's main branch has diverged from upstream/main
// (no conflicts, but cannot fast-forward), the tool shows helpful error message with
// instructions to reset the main branch to match upstream
func TestDevD_MainBranchDiverged(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"-d"},
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateOK,
		SyncState:         systrun.SyncStateMainBranchDiverged,
		NeedCollaboration: true,
		ExpectedStdout: []string{
			"Error: Cannot fast-forward merge upstream/main into main", // Part of MsgCannotFastForward
			gitcmds.MsgMainBranchDiverged,
			gitcmds.MsgToFixRunCommands,
			gitcmds.MsgGitCheckoutMain,
			gitcmds.MsgGitFetchUpstream,
			gitcmds.MsgGitResetHardUpstream,
			gitcmds.MsgGitPushOriginMainForce,
			"Warning: This will overwrite your main branch", // Part of MsgWarningOverwriteMainBranch
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	// The command should fail with an error because fast-forward is not possible
	require.Error(err)
}

// getGithubConfig retrieves GitHub credentials from environment variables
// and skips the test if any credentials are missing
// TestQS_FilesWithSpaces tests that qs can handle files with spaces in their names
func TestQS_FilesWithSpaces(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateOK,
		SyncState:         systrun.SyncStateUnspecified,
		ClipboardContent:  systrun.ClipboardContentGithubIssue,
		NeedCollaboration: true,
		Expectations: []systrun.ExpectationFunc{
			systrun.ExpectationFilesWithSpacesHandled,
		},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

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

// TestPR_CustomText tests creating a PR from custom text (qs dev "some text" -> qs pr)
// Verifies that the PR title and commit message use the custom text without prompting
func TestPR_CustomText(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "pr",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateOK,
		SyncState:         systrun.SyncStateSynchronized,
		ClipboardContent:  systrun.ClipboardContentCustom,
		NeedCollaboration: true,
		Expectations:      []systrun.ExpectationFunc{systrun.ExpectationPRCreated},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}

// TestPR_CustomText_NoUpstream tests creating a PR from custom text in single remote mode
// Verifies that the PR title and commit message use the custom text without prompting
func TestPR_CustomText_NoUpstream(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
			Command: "pr",
		},
		UpstreamState:     systrun.RemoteStateOK,
		ForkState:         systrun.RemoteStateNull,
		SyncState:         systrun.SyncStateSynchronized,
		ClipboardContent:  systrun.ClipboardContentCustom,
		NeedCollaboration: true,
		Expectations:      []systrun.ExpectationFunc{systrun.ExpectationPRCreated},
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.NoError(err)
}
