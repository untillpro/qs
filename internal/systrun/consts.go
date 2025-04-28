package systrun

const (
	TestDataDir     = "./.testdata"
	GithubURL       = "https://github.com"
	defaultDirPerms = 0755
)

const (
	EnvForkGithubAccount     = "GH_ACCOUNT"
	EnvForkGithubToken       = "GH_TOKEN"
	EnvUpstreamGithubAccount = "UPSTREAM_GH_ACCOUNT"
	EnvUpstreamGithubToken   = "UPSTREAM_GH_TOKEN"
)

type RemoteState int
type SyncState int

const (
	// RemoteStateOK means that remote of the clone repo is configured correctly
	RemoteStateOK RemoteState = iota
	// RemoteStateMisconfigured means that the remote of the clone repo is not configured correctly
	RemoteStateMisconfigured
	// RemoteStateNull means that the remote of the clone repo is null
	RemoteStateNull
)

const (
	// SyncStateSynchronized means that the clone repo and fork/upstream repos are in sync
	SyncStateSynchronized SyncState = iota
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
)
