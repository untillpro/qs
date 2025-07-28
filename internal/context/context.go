package context

type CtxKey int

const (
	CtxKeyCloneRepoPath CtxKey = iota
	CtxKeyAnotherCloneRepoPath
	CtxKeyDevBranchName
	CtxKeyCustomBranchName
	CtxKeyCreatedGithubIssueURL
	CtxKeyJiraTicket
	CtxKeyBranchPrefix
	CtxKeyClipboard
	CtxKeyCommitMessage
)
