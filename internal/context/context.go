package context

type CtxKey int

const (
	CtxKeyCloneRepoPath CtxKey = iota
	CtxKeyAnotherCloneRepoPath
	CtxKeyDevBranchName
	CtxKeyCustomBranchName
	CtxKeyCreatedGithubIssueURL
	CtxKeyBranchPrefix
	CtxKeyClipboard
	CtxKeyCommitMessage
)
