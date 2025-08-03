package systrun

const (
	TestDataDir                 = ".testdata"
	GithubURL                   = "https://github.com"
	remoteGithubRepoURLTemplate = GithubURL + "/%s/%s.git"
	issueBody                   = "Automated test issue created by QS system test framework"
	origin                      = "origin"
	upstream                    = "upstream"
	git                         = "git"
	errFormatFailedToCloneRepos = "failed to open cloned repository: %w"
	formatGithubTokenEnv        = "GITHUB_TOKEN=%s"
	cloneRepoDirPerm            = 0755
	commitFilePerm              = 0644
	readmeMDFileName            = "README.md"
	changeDirFlag               = "-C"
	defaultDevBranchName        = "branch-name-dev"
	defaultPrBranchName         = "branch-name-pr"
)

const (
	EnvForkGithubAccount     = "FORK_GH_ACCOUNT"
	EnvForkGithubToken       = "FORK_GH_TOKEN"
	EnvUpstreamGithubAccount = "UPSTREAM_GH_ACCOUNT"
	EnvUpstreamGithubToken   = "UPSTREAM_GH_TOKEN"
)

type RemoteState int

// e.g. if SyncState is SyncStateSynchronized then do nothing more
// e.g. if SyncStateForkChanged then additionally one push from another clone
type SyncState int
type ClipboardContentType int
type DevBranchState int

const (
	// RemoteStateOK means that remote of the clone repo is configured correctly
	RemoteStateOK RemoteState = iota
	// RemoteStateMisconfigured means that the remote of the clone repo is not configured correctly,
	// e.g. `qs u` should fail on permission error on `git push` (now it does not fail)
	RemoteStateMisconfigured
	// RemoteStateNull means that the remote of the clone repo is null
	RemoteStateNull
)

const (
	SyncStateUnspecified SyncState = iota
	// SyncStateUncommitedChangesInClone means that the clone repo has uncommitted changes
	SyncStateUncommitedChangesInClone
	// SyncStateSynchronized means that the clone repo and fork/upstream repos are in sync
	SyncStateSynchronized
	// SyncStateForkChanged means that the fork repo has changes that are not in the clone repo
	SyncStateForkChanged
	// SyncStateCloneChanged means that the clone repo has changes that are not in the fork repo
	SyncStateCloneChanged
	// SyncStateBothChanged means that both the clone and fork repos have changes
	SyncStateBothChanged
	// SyncStateBothChangedConflict means that both the clone and fork repos have changes that conflict with each other
	SyncStateBothChangedConflict
	// SyncStateDoesntTrackOrigin means that the dev branch does not track the origin
	SyncStateDoesntTrackOrigin
	// SyncStateCloneIsAheadOfFork means that the clone repo is ahead of the fork repo
	SyncStateCloneIsAheadOfFork
)

const (
	// Empty clipboard content before running the test command
	ClipboardContentEmpty ClipboardContentType = iota
	// GitHub issue content to be set in the clipboard before running the test command
	ClipboardContentGithubIssue
	// Unavailable GitHub issue content to be set in the clipboard before running the test command
	ClipboardContentUnavailableGithubIssue
	// Jira ticket content to be set in the clipboard before running the test command
	ClipboardContentJiraTicket
	// Custom content to be set in the clipboard before running the test command
	ClipboardContentCustom
)

const (
	// Dev branch does not exist
	DevBranchStateNotExists DevBranchState = iota
	// Dev branch exists and it is the current branch
	DevBranchStateExistsAndCheckedOut
	// Dev branch exists but it is not the current branch
	DevBranchStateExistsButNotCheckedOut
)

const (
	headerOfFilesInAnotherClone = "another clone header"
)

func (c ClipboardContentType) String() string {
	switch c {
	case ClipboardContentEmpty:
		return "ClipboardContentEmpty"
	case ClipboardContentGithubIssue:
		return "ClipboardContentGithubIssue"
	case ClipboardContentUnavailableGithubIssue:
		return "ClipboardContentUnavailableGithubIssue"
	case ClipboardContentJiraTicket:
		return "ClipboardContentJiraTicket"
	case ClipboardContentCustom:
		return "ClipboardContentCustom"
	default:
		return "unknown"
	}
}

func (s SyncState) String() string {
	switch s {
	case SyncStateUnspecified:
		return "SyncStateUnspecified"
	case SyncStateUncommitedChangesInClone:
		return "SyncStateUncommitedChangesInClone"
	case SyncStateSynchronized:
		return "SyncStateSynchronized"
	case SyncStateForkChanged:
		return "SyncStateForkChanged"
	case SyncStateCloneChanged:
		return "SyncStateCloneChanged"
	case SyncStateBothChanged:
		return "SyncStateBothChanged"
	case SyncStateBothChangedConflict:
		return "SyncStateBothChangedConflict"
	case SyncStateDoesntTrackOrigin:
		return "SyncStateDoesntTrackOrigin"
	default:
		return "unknown"
	}
}
