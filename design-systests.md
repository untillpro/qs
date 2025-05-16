# Design of the system tests for the `qs` utility

## Introduction

System tests are designed to validate the functionality of the `qs` utility, to ensure that the utility behaves as expected in different situations.

## Motivation

The motivation behind these system tests is to ensure that the `qs` utility works correctly in a real-world scenario, where it interacts with remote repositories on GitHub. The tests will cover various aspects of the utility, including forking repositories, creating branches, making commits, and creating pull requests.

## Definitions:

- **Issue**: it is a Github-issue
- **Upstream Repo**: the original GitHub repository (often the main project you forked from)
- **Fork Repo**: it is copy of the upstream repo under your GitHub account. Lives on the GitHub server
- **Clone Repo**: it is repo on PC cloned from your fork repo
- **Upstream Github Account**: it is a Github-account used for access for Upstream Repo. It will be provided with a token that has access to the Upstream Repo
- **Fork Github Account**: it is a Github-account used for access for Fork Repo. It will be provided with a token that has access to the Fork Repo
- **Remote**: it is a reference to some repo we can push to or pull from. It is a URL of the repo on GitHub
   - it is not a branch  
- **Upstream Remote**: it is remote leading to `Upstream Repo`. Lives in the clone repo
- **Origin Remote**: it is remote leading to Fork repo. Lives in the clone repo
- **Fork Branch**: it is a branch in fork repo. Lives on the GitHub server
- **Fork-tracking Branch**: it is local reference to the latest known state of the Fork branch
  - Exists as refs/remotes/origin/branch_name in the clone repo
- **Main Branch**: it is a `master` or `main`
- **Working Copy**: it is a directory on the local machine which represents state of the clone repo on current commit of the Local branch and could contain new modifications
- **Test Framework**: it is a set of scripts and methods used to run system tests. It will be responsible for creating and deleting test repos, running the `qs` utility, and checking the results
- **Test Environment**: it is a set of upstream repo, fork repo, clone repo, values of upstream and origin
- **System Test**: it is a test written in Go language, which will be executed by test framework. It will use the `qs` utility to perform various operations on the test environment and check the results

## Principles

- The framework must check prerequisites: existence of `qs` utility, `gh` utility, and `git` utility;
- One system test - one `qs` command
- Flow:
  - Build the environment according to TestConfig
  - Run the command
  - Validate the command output using ExpectedStderr and ExpectedStdout
  - Validate command result via checking expectations
- RepoName: TestID-YYMMDDhhmmss
  - TestID is a unique identifier for the test
  - YYMMDDhhmmss is the date and time when the test was run
- UPSTREAM_GH_ACCOUNT: it is environment variable used to set the GitHub account for the `Upstream Repo`
- UPSTREAM_GH_TOKEN: it is environment variable used to set the token for GitHub account of the `Upstream Repo`
- FORK_GH_ACCOUNT: it is environment variable used to set the GitHub account for the `Fork Repo`
- FORK_GH_TOKEN: it is environment variable used to set the token for GitHub account of the `Fork Repo`
- Upstream repo path: github.com/{{.UPSTREAM_GH_ACCOUNT }}/{{.RepoName}}
- Fork repo path: github.com/{{.FORK_GH_ACCOUNT}}/{{.RepoName}}
- Clone repo path: ./.testdata/RepoName
- Use cases:
  - qs fork:
    UpstreamState{OK, Misconfigured, Null}, ForkState{OK, Misconfigured, Null} If ForkState == Null && UpstreamState == OK {remotes.origin = UPSTREAM_REPO_URL }
    +-----------------------------+---------------+-----------+---------------------------------------------------------------------------------------------+
    | Name                        | UpstreamState | ForkState | Expected Output                                                                             |
    +-----------------------------+---------------+-----------+---------------------------------------------------------------------------------------------+
    | Fork does not exist         | OK            | Null      | Fork repo created                                                                           |
    | Fork exists                 | OK            | OK        | Adjusted remotes of the clone repo (origin → fork, upstream → upstream)                     |
    | No origin remote            | Null          | Null      | Error message: "origin remote not found"                                                    |
    +-----------------------------+---------------+-----------+---------------------------------------------------------------------------------------------+
  - qs dev:
    UpstreamState{OK, Misconfigured, Null}, ForkState{OK, Misconfigured, Null}, DevBranchExists                                                                                                             | Expected Output                                            |
    +-----------------------------+---------------+-----------+-----------------+---------------------------------------------------------------------------+
    | Name                        | UpstreamState | ForkState | DevBranchExists | Expected Output                                                           |
    +-----------------------------+---------------+-----------+-----------------+---------------------------------------------------------------------------+
    | New dev branch needed       | OK            | OK        | false           | New dev branch is created in both clone and fork repos                    |
    | Dev branch exists           | OK            | OK        | true            | New dev branch is not created                                             |
    | Fork missing, branch missing| OK            | Null      | false           | New dev branch is created in both clone and upstream repos                |
    | Fork missing, branch exists | OK            | Null      | true            | New dev branch is not created                                             |
    +-----------------------------+---------------+-----------+-----------------+---------------------------------------------------------------------------+
  - qs pr:
    UpstreamState{OK, Misconfigured, Null}, ForkState{OK, Misconfigured, Null}, SyncState{Synchronized, ForkChanged, CloneChanged, BothChanged, BothChangedConflict, DoesntTrackOrigin} DoesntTrackOrigin means state when dev branch tracks non-origin remote 
    +-----------------------------+---------------+-----------+-------------------+-------------------------------------------------------------------------+
    | Name                        | UpstreamState | ForkState | SyncState         | Expected Output                                                         |
    +-----------------------------+---------------+-----------+-------------------+-------------------------------------------------------------------------+
    | Basic                       | OK            | OK        | Synchronized      | New pull request is created in upstream repo                            |
    | Upstream missing            | OK            | Null      | Synchronized      | New pull request is created in upstream repo                            |
    | Dev branch out of date      | OK            | OK        | ForkChanged       | Error message: "This branch is out-of-date. Merge automatically [y/n]?" |
    | Wrong branch checked out    | OK            | OK        | DoesntTrackOrigin | Error message: "You are not on dev branch"                              |
    +-----------------------------+---------------+-----------+-------------------+-------------------------------------------------------------------------+    
  - qs d
  - qs u


## Implementation

### File structure

1. `./internal/systrun/provide.go`
```go
func New(t *testing.T, testConfig *TestConfig) *SystemTest {
    cloneRepoPath := //generate a unique path for the clone repo in .testdata dir of the root of the qs package
    
	return &SystemTest{t: t, cfg: testConfig, cloneRepoPath: cloneRepoPath}
}
```

2. `./internal/systrun/impl.go`
```go
func (st *SystemTest) Run() error {
    // Check prerequisites
    if err := st.checkPrerequisites(); err != nil {
        return err
    }

    // Create test environment
    if err := st.createTestEnvironment(); err != nil {
        return err
    }

    // Run the command
    actualStdout, actualStderr, err := st.runCommand();
    if err != nil {
        return err
    }

    // Validate output
    if err := st.validateOutput(actualStdout, actualStderr); err != nil {
        return err
    }

  // Check expectations
  if err := st.checkExpectations(); err != nil {
      return err
  }

    return nil
}
```

3. `./internal/systrun/types.go`

```go
type SystemTest struct {
	t   *testing.T
	cfg *TestConfig
}

// TestConfig contains all configuration for a system test
type TestConfig struct {
	TestID          string
    GHConfig        GithubConfig
    Command         string
    Args            []string
    UpstreamState   RemoteState
    ForkState       RemoteState
    SyncState       SyncState
    DevBranchExists bool
    ExpectedStderr  string
    ExpectedStdout  string
    Expectations    []IExpectation
}

// GithubConfig holds GitHub account and token information
type GithubConfig struct {
	UpstreamAccount string
    UpstreamToken   string
    ForkAccount     string
    ForkToken       string
}

// ExpectedDevBranch represents the expected state of the dev branch
type ExpectedDevBranch struct {
    Exists bool
}

func (e ExpectedDevBranch) Check(cloneRepoPath string) error {
    // Check if the dev branch exists in the clone repo
	
    return nil
}

// ExpectedRemoteState represents the expected state of a remote
type ExpectedRemoteState struct {
    UpstreamRemoteState RemoteState
    ForkRemoteState     RemoteState
}

func (e ExpectedRemoteState) Check(cloneRepoPath string) error {
    // Implement the logic to check the remote state

    return nil
}

// ExpectedPullRequest represents the expected state of a pull request
type ExpectedPullRequest struct {
    Exists          bool
    Title           string
    ForkAccount     string
    UpstreamAccount string
}

func (e ExpectedPullRequest) Check(cloneRepoPath string) error {
	// Check if the pull request exists in the upstream repo
	
    return nil
}

// ExpectedDownloadResult represents the expected state after downloading changes
type ExpectedDownloadResult struct {}

func (e ExpectedDownloadResult) Check(cloneRepoPath string) error {
    // Compare local and remote branches to ensure they're in sync
    
	return nil
}

// ExpectedUploadResult represents the expected state after uploading changes
type ExpectedUploadResult struct {}

func (e ExpectedUploadResult) Check(cloneRepoPath string) error {
	// Compare local and remote branches to ensure they're in sync
	
    return nil
}
```

4. `./internal/systrun/consts.go`
```go
type RemoteState int
type SyncState int

const (
	RemoteStateOK RemoteState = iota
	RemoteStateMisconfigured
	RemoteStateNull
)

const (
	SyncStateSynchronized SyncState = iota
	SyncStateForkChanged
	SyncStateCloneChanged
	SyncStateBothChanged
	SyncStateBothChangedConflict
	SyncStateDoesntTrackOrigin
)
```

5. `./internal/systrun/impl.go`
```go
// checkExpectations checks expectations after command execution
func (st *SystemTest) checkExpectations() error {
  for _, expectation := range st.cfg.Expectations {
    if err := expectation.Check(st.cloneRepoPath); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
  }
  
  return nil
}

func (st *SystemTest) createTestEnvironment() error {
    // create upstream, form and clone repos according to the test config
}
```

6. `./internal/systrun/interface.go`
```go
// IExpectation is an interface for expectations in system tests
type IExpectation interface {
	// Check compares the current state of the system with the expected state
	Check(cloneRepoPath string) error
}
```

