/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package utils

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
