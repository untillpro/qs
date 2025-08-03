# Design of the system tests for the `qs` utility

## Introduction

System tests are designed to validate the functionality of the `qs` utility, to ensure that the utility behaves as expected in different situations. The system tests create real GitHub repositories, perform actual Git operations, and validate the results against expected outcomes.

## Motivation

The motivation behind these system tests is to ensure that the `qs` utility works correctly in a real-world scenario, where it interacts with remote repositories on GitHub. The tests cover various aspects of the utility, including:
- Forking repositories and configuring remotes
- Creating development and PR branches
- Making commits and uploading changes
- Creating pull requests and managing branch lifecycle
- Downloading changes and synchronizing repositories
- Integration with GitHub issues and Jira tickets

## Definitions

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

- **Prerequisites**: The framework checks for existence of `qs` utility, `gh` utility, and `git` utility
- **One test, one command**: Each system test focuses on testing a single `qs` command
- **Real GitHub integration**: Tests create actual GitHub repositories and perform real Git operations
- **Comprehensive validation**: Tests validate both command output and resulting repository state
- **Automatic cleanup**: Test repositories are automatically created and cleaned up
- **Retry mechanisms**: Network operations include retry logic for reliability
- **Cross-platform support**: Tests run on Windows, Linux, and macOS

### Test Flow:
1. **Check prerequisites** - Verify required tools are available
2. **Create test environment** - Set up upstream, fork, and clone repositories
3. **Configure test state** - Set up branches, remotes, and sync states as needed
4. **Run the command** - Execute the specified `qs` command
5. **Validate output** - Check stdout/stderr against expected patterns
6. **Check expectations** - Verify repository state matches expected outcomes
7. **Cleanup** - Remove test repositories and temporary files

### Repository Naming:
- **Pattern**: `TestID-OS-YYMMDDhhmmss`
  - `TestID`: Unique identifier for the test (e.g., "testfork_onexistingfork")
  - `OS`: Operating system (darwin, linux, windows)
  - `YYMMDDhhmmss`: Timestamp when the test was run

### Environment Variables:
- `UPSTREAM_GH_ACCOUNT`: GitHub account for the upstream repository
- `UPSTREAM_GH_TOKEN`: GitHub Personal Access Token for upstream account
- `FORK_GH_ACCOUNT`: GitHub account for the fork repository
- `FORK_GH_TOKEN`: GitHub Personal Access Token for fork account
- `JIRA_API_TOKEN`: Jira API token (for Jira integration tests)
- `JIRA_TICKET_URL`: Jira ticket URL (for Jira integration tests)
- `JIRA_EMAIL`: Jira email (for Jira integration tests)

### Repository Paths:
- **Upstream repo**: `github.com/{{UPSTREAM_GH_ACCOUNT}}/{{RepoName}}`
- **Fork repo**: `github.com/{{FORK_GH_ACCOUNT}}/{{RepoName}}`
- **Clone repo**: `./.testdata/{{RepoName}}`
- **Another clone**: `./.testdata/{{RepoName}}-another` (for multi-clone tests)

## Test Scenarios

### qs fork

Tests the repository forking functionality and remote configuration.

| Test Case                    | UpstreamState | ForkState | Expected Behavior                                                       |
|------------------------------|---------------|-----------|-------------------------------------------------------------------------|
| **Fork does not exist**     | OK            | Null      | Fork repo created, remotes configured (origin → fork, upstream → upstream) |
| **Fork already exists**     | OK            | OK        | Error: "you are in fork already"                                       |
| **No origin remote**        | Null          | Null      | Error: "origin remote not found"                                       |

### qs dev

Tests development branch creation and management.

| Test Case                    | UpstreamState | ForkState | DevBranchState | Expected Behavior                                          |
|------------------------------|---------------|-----------|----------------|------------------------------------------------------------|
| **New dev branch**           | OK            | OK        | NotExists      | New dev branch created in clone and pushed to fork        |
| **Dev branch exists**        | OK            | OK        | ExistsAndCheckedOut | Switch to existing dev branch                      |
| **Fork missing**             | OK            | Null      | NotExists      | New dev branch created in clone and pushed to upstream    |
| **Custom branch name**       | OK            | OK        | NotExists      | Branch created with custom name from clipboard/input      |
| **Delete dev branch (-d)**   | OK            | OK        | ExistsAndCheckedOut | Dev branch deleted locally and remotely            |

### qs pr

Tests pull request creation and branch lifecycle management.

| Test Case                    | UpstreamState | ForkState | SyncState         | Expected Behavior                                          |
|------------------------------|---------------|-----------|-------------------|------------------------------------------------------------|
| **Basic PR creation**        | OK            | OK        | Synchronized      | PR created, dev branch → pr branch, dev branch deleted    |
| **Fork missing**             | OK            | Null      | Synchronized      | PR created directly to upstream                            |
| **Branch out of sync**       | OK            | OK        | ForkChanged       | Error or automatic merge prompt                            |
| **Wrong branch**             | OK            | OK        | DoesntTrackOrigin | Error: branch not properly configured                      |

### qs u (upload)

Tests uploading local changes to remote repositories.

| Test Case                    | UpstreamState | ForkState | SyncState                    | Expected Behavior                                          |
|------------------------------|---------------|-----------|------------------------------|------------------------------------------------------------|
| **Upload changes**           | OK            | OK        | UncommittedChangesInClone    | Changes committed and pushed to fork                       |
| **No changes**               | OK            | OK        | Synchronized                 | No operation needed                                        |
| **First push**               | OK            | OK        | CloneChanged                 | Tracking branch set up, changes pushed                    |

### qs d (download)

Tests downloading and synchronizing changes from remote repositories.

| Test Case                    | UpstreamState | ForkState | SyncState         | Expected Behavior                                          |
|------------------------------|---------------|-----------|-------------------|------------------------------------------------------------|
| **Download changes**         | OK            | OK        | ForkChanged       | Local repo synchronized with fork and upstream            |
| **No changes**               | OK            | OK        | Synchronized      | No operation needed                                        |
| **Upstream only**            | OK            | Null      | Synchronized      | Local repo synchronized with upstream                      |

## Architecture

### Core Components

#### 1. SystemTest (`./internal/systrun/types.go`)

```go
type SystemTest struct {
    ctx                  context.Context
    cfg                  *TestConfig
    cloneRepoPath        string
    anotherCloneRepoPath string
    repoName             string
    qsExecRootCmd        func(ctx context.Context, args []string) (context.Context, error)
}
```

The main test execution engine that orchestrates the entire test lifecycle.

#### 2. Test Factory (`./internal/systrun/provide.go`)

```go
func New(t *testing.T, testConfig *TestConfig) *SystemTest {
    timestamp := time.Now().Format("060102150405") // YYMMDDhhmmss
    repoName := fmt.Sprintf("%s-%s-%s", testConfig.TestID, runtime.GOOS, timestamp)

    return &SystemTest{
        ctx:           context.Background(),
        cfg:           testConfig,
        repoName:      repoName,
        cloneRepoPath: filepath.Join(wd, TestDataDir, repoName),
        qsExecRootCmd: cmdproc.ExecRootCmd,
    }
}
```

Creates unique test instances with OS-specific repository names.

#### 3. Test Execution Flow (`./internal/systrun/impl.go`)

```go
func (st *SystemTest) Run() error {
    // 1. Check prerequisites (qs, gh, git availability)
    if err := st.checkPrerequisites(); err != nil {
        return err
    }

    // 2. Create test environment (repos, remotes, branches)
    if err := st.createTestEnvironment(); err != nil {
        return err
    }

    // 3. Set GitHub authentication
    if err := os.Setenv("GITHUB_TOKEN", st.cfg.GHConfig.ForkToken); err != nil {
        return err
    }

    // 4. Execute the qs command
    stdout, stderr, err := st.runCommand(st.cfg.CommandConfig)

    // 5. Validate command output
    if err != nil {
        if err := st.validateStderr(stderr); err != nil {
            return err
        }
    }
    if err := st.validateStdout(stdout); err != nil {
        return err
    }

    // 6. Check post-execution expectations
    if err := st.checkExpectations(); err != nil {
        return err
    }

    // 7. Cleanup test environment
    if err := st.cleanupTestEnvironment(); err != nil {
        return err
    }

    return err
}
```

### Configuration Types (`./internal/systrun/types.go`)

#### TestConfig Structure

```go
type TestConfig struct {
    TestID                 string                // Unique test identifier
    GHConfig               GithubConfig          // GitHub authentication
    CommandConfig          *CommandConfig        // Command to execute
    UpstreamState          RemoteState           // Upstream repo state
    ForkState              RemoteState           // Fork repo state
    SyncState              SyncState             // Synchronization state
    DevBranchState         DevBranchState        // Dev branch state
    ClipboardContent       ClipboardContentType  // Clipboard content type
    RunCommandOnOtherClone bool                  // Run on secondary clone
    NeedCollaboration      bool                  // Setup collaboration
    BranchState            *BranchState          // Complex branch states
    ExpectedStderr         string                // Expected error output
    ExpectedStdout         string                // Expected standard output
    Expectations           []ExpectationFunc     // Post-execution checks
}

type CommandConfig struct {
    Command string   // qs command name (fork, dev, pr, u, d)
    Args    []string // Command arguments
    Stdin   string   // Input to provide to command
}

type GithubConfig struct {
    UpstreamAccount string // GitHub username for upstream
    UpstreamToken   string // GitHub token for upstream
    ForkAccount     string // GitHub username for fork
    ForkToken       string // GitHub token for fork
}

type BranchState struct {
    DevBranchExists      bool // Dev branch exists locally/remotely
    DevBranchHasRtBranch bool // Dev branch has remote tracking
    DevBranchIsAhead     bool // Dev branch ahead of remote
    PRBranchExists       bool // PR branch exists
    PRBranchHasRtBranch  bool // PR branch has remote tracking
    PRBranchIsAhead      bool // PR branch ahead of remote
    PRExists             bool // Pull request exists on GitHub
    PRMerged             bool // Pull request is merged
}
```

#### State Enumerations (`./internal/systrun/const.go`)

```go
type RemoteState int
const (
    RemoteStateOK RemoteState = iota           // Remote configured correctly
    RemoteStateMisconfigured                   // Remote misconfigured
    RemoteStateNull                            // No remote configured
)

type SyncState int
const (
    SyncStateUnspecified SyncState = iota      // Not specified
    SyncStateUncommitedChangesInClone          // Uncommitted local changes
    SyncStateSynchronized                      // All repos in sync
    SyncStateForkChanged                       // Fork has new changes
    SyncStateCloneChanged                      // Clone has new changes
    SyncStateBothChanged                       // Both have changes
    SyncStateBothChangedConflict               // Conflicting changes
    SyncStateDoesntTrackOrigin                 // Branch doesn't track origin
)

type DevBranchState int
const (
    DevBranchStateNotExists DevBranchState = iota     // Branch doesn't exist
    DevBranchStateExistsAndCheckedOut                 // Branch exists and current
    DevBranchStateExistsButNotCheckedOut              // Branch exists but not current
)

type ClipboardContentType int
const (
    ClipboardContentEmpty ClipboardContentType = iota // Empty clipboard
    ClipboardContentGithubIssue                       // GitHub issue URL
    ClipboardContentUnavailableGithubIssue            // Invalid GitHub issue
    ClipboardContentJiraTicket                        // Jira ticket URL
    ClipboardContentCustom                            // Custom content
)
```

### Expectation System (`./internal/systrun/types.go`)

The expectation system validates the post-execution state of repositories and GitHub resources.

#### Expectation Function Type

```go
type ExpectationFunc func(ctx context.Context) error
```

#### Built-in Expectations

```go
// Repository State Expectations
func ExpectationForkExists(ctx context.Context) error
func ExpectationCurrentBranchHasPrefix(ctx context.Context) error
func ExpectationCustomBranchIsCurrentBranch(ctx context.Context) error
func ExpectationBranchLinkedToIssue(ctx context.Context) error

// Branch Count Expectations
func ExpectationOneLocalBranch(ctx context.Context) error
func ExpectationTwoLocalBranches(ctx context.Context) error
func ExpectationThreeLocalBranches(ctx context.Context) error
func ExpectationOneRemoteBranch(ctx context.Context) error
func ExpectationTwoRemoteBranches(ctx context.Context) error
func ExpectationThreeRemoteBranches(ctx context.Context) error

// Pull Request Expectations
func ExpectationPRCreated(ctx context.Context) error

// Synchronization Expectations
func ExpectationCloneIsSyncedWithFork(ctx context.Context) error
func ExpectationRemoteBranchWithCommitMessage(ctx context.Context) error
func ExpectationNotesDownloaded(ctx context.Context) error

// Tool Integration Expectations
func ExpectationLargeFileHooksInstalled(ctx context.Context) error
```

#### Context-Based Data Passing

The system uses Go's `context.Context` to pass data between test phases:

```go
// Context Keys (from internal/context package)
CtxKeyCloneRepoPath         // Path to main clone
CtxKeyAnotherCloneRepoPath  // Path to secondary clone
CtxKeyDevBranchName         // Name of dev branch
CtxKeyCustomBranchName      // Custom branch name
CtxKeyBranchPrefix          // Branch prefix
CtxKeyCreatedGithubIssueURL // Created GitHub issue URL
CtxKeyJiraTicket            // Jira ticket identifier
CtxKeyCommitMessage         // Commit message parts
```

### Test Environment Setup (`./internal/systrun/impl.go`)

#### Environment Creation Process

```go
func (st *SystemTest) createTestEnvironment() error {
    // 1. Create upstream repository on GitHub
    if err := st.createUpstreamRepo(st.repoName, repoURL); err != nil {
        return err
    }

    // 2. Create fork repository (if needed)
    if err := st.createForkRepo(st.repoName); err != nil {
        return err
    }

    // 3. Clone repository locally
    if err := st.cloneRepo(cloneURL, st.cloneRepoPath, authToken); err != nil {
        return err
    }

    // 4. Configure remotes based on test scenario
    if err := st.configureRemotes(st.cloneRepoPath, st.repoName); err != nil {
        return err
    }

    // 5. Setup dev branch if needed
    if err := st.setupDevBranch(); err != nil {
        return err
    }

    // 6. Configure collaboration (if needed)
    if st.cfg.NeedCollaboration {
        if err := st.configureCollaboration(); err != nil {
            return err
        }
    }

    // 7. Process clipboard content
    if err := st.processClipboardContent(); err != nil {
        return err
    }

    // 8. Setup sync state
    if err := st.processSyncState(); err != nil {
        return err
    }

    // 9. Create additional clone (if needed)
    if err := st.createAnotherClone(); err != nil {
        return err
    }

    return nil
}
```

## Test Examples

### Basic Fork Test

```go
func TestFork_OnExistingFork(t *testing.T) {
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
    require.Error(t, err) // Expect error for existing fork
}
```

### Complex Dev Branch Test

```go
func TestDevD_DevBranch_RT_PRMerged(t *testing.T) {
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
    require.NoError(t, err)
}
```

### Upload Test with Sync State

```go
func TestUpload(t *testing.T) {
    testConfig := &systrun.TestConfig{
        TestID:   strings.ToLower(t.Name()),
        GHConfig: getGithubConfig(t),
        CommandConfig: &systrun.CommandConfig{
            Command: "u",
        },
        UpstreamState:     systrun.RemoteStateOK,
        ForkState:         systrun.RemoteStateOK,
        SyncState:         systrun.SyncStateUncommitedChangesInClone,
        ClipboardContent:  systrun.ClipboardContentGithubIssue,
        NeedCollaboration: true,
        Expectations:      []systrun.ExpectationFunc{
            systrun.ExpectationRemoteBranchWithCommitMessage,
        },
    }

    sysTest := systrun.New(t, testConfig)
    err := sysTest.Run()
    require.NoError(t, err)
}
```

## Key Features

### 1. **Retry Mechanisms**
- All network operations include retry logic for reliability
- Configurable via environment variables:
  - `QS_MAX_RETRIES`: Maximum retry attempts (default: 3)
  - `QS_RETRY_DELAY_SECONDS`: Initial delay (default: 2)
  - `QS_MAX_RETRY_DELAY_SECONDS`: Maximum delay (default: 30)

### 2. **Cross-Platform Support**
- Tests run on Windows, Linux, and macOS
- OS-specific repository naming prevents conflicts
- Platform-specific dependency installation in CI

### 3. **Real GitHub Integration**
- Creates actual GitHub repositories
- Tests real GitHub API interactions
- Validates GitHub CLI functionality
- Supports GitHub issue and Jira ticket integration

### 4. **Comprehensive Validation**
- Command output validation (stdout/stderr)
- Repository state validation
- Branch existence and configuration
- Remote tracking branch validation
- Pull request creation validation
- Notes synchronization validation

### 5. **Automatic Cleanup**
- Test repositories automatically deleted after execution
- Temporary files cleaned up
- GitHub resources properly removed

## Best Practices

### Test Design
1. **One command per test** - Each test focuses on a single `qs` command
2. **Clear naming** - Test names reflect the scenario being tested
3. **Comprehensive expectations** - Validate both success and failure cases
4. **Realistic scenarios** - Test real-world usage patterns

### Configuration
1. **Use environment variables** - Store sensitive data in GitHub secrets
2. **Unique test IDs** - Ensure test isolation with unique identifiers
3. **Proper state setup** - Configure initial state to match test scenario
4. **Clear expectations** - Define specific, measurable outcomes

### Debugging
1. **Verbose logging** - Use `-v` flag for detailed output
2. **Context inspection** - Examine context values for debugging
3. **Repository inspection** - Check `.testdata` directory for repo state
4. **GitHub inspection** - Verify GitHub resources manually if needed

## CI/CD Integration

### GitHub Actions Workflow
- Runs on Windows, Linux, and macOS
- Uses repository secrets for authentication
- Installs platform-specific dependencies
- Runs tests with proper environment variables
- Provides detailed failure reporting

### Required Secrets
- `UPSTREAM_GH_ACCOUNT` / `UPSTREAM_GH_TOKEN`
- `FORK_GH_ACCOUNT` / `FORK_GH_TOKEN`
- `JIRA_API_TOKEN` / `JIRA_TICKET_URL` / `JIRA_EMAIL` (optional)

### Environment Variables
- Retry configuration variables
- `QS_SKIP_QS_VERSION_CHECK=true` for CI environments
```
