package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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

// TestDev_NoUpstream_CustomName tests creating a new dev branch when it doesn't exist
func TestDev_NoUpstream_CustomName(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
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

// TestDevNoForkExistingIssue tests creating a dev branch when upstream remote doesn't exist
func TestDev_NoFork_ExistingIssue(t *testing.T) {
	require := require.New(t)

	ghConfig := getGithubConfig(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: ghConfig,
		CommandConfig: &systrun.CommandConfig{
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

// TestDev_NoFork_NonExistingIssue tests creating a dev branch when upstream remote doesn't exist
func TestDev_NoFork_NonExistingIssue(t *testing.T) {
	require := require.New(t)

	ghConfig := getGithubConfig(t)
	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: ghConfig,
		CommandConfig: &systrun.CommandConfig{
			Command: "dev",
			Args:    []string{"--no-fork"},
			Stdin:   "y",
		},
		ClipboardContent: systrun.ClipboardContentUnavailableGithubIssue,
		UpstreamState:    systrun.RemoteStateOK,
		ForkState:        systrun.RemoteStateNull,
		ExpectedStderr:   "invalid GitHub issue link",
	}

	sysTest := systrun.New(t, testConfig)
	err := sysTest.Run()
	require.Error(err)
}

// TestDevNoForkJiraTicketURL tests creating a dev branch with a valid JIRA ticket URL
func TestDev_NoFork_JiraTicketURL(t *testing.T) {
	require := require.New(t)

	testConfig := &systrun.TestConfig{
		TestID:   strings.ToLower(t.Name()),
		GHConfig: getGithubConfig(t),
		CommandConfig: &systrun.CommandConfig{
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
		ForkState: systrun.RemoteStateOK,
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
		ForkState: systrun.RemoteStateOK,
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
			systrun.ExpectationLargeFileHookFunctional,
		},
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
