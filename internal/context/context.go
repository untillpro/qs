package context

type CtxKey int

const (
	CtxKeyCloneRepoPath CtxKey = iota
	CtxKeyDevBranchName
	CtxKeyCustomBranchName
	CtxKeyCreatedGithubIssueURL
	CtxKeyBranchPrefix
)
