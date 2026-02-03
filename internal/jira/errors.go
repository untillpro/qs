package jira

import "errors"

var (
	ErrJiraIssueNotFoundOrInsufficientPermission = errors.New("issue does not exist or you do not have permission to see it")
)
