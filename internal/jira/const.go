package jira

const (
	jiraDomain = "https://untill.atlassian.net"

	NotFoundIssueOrInsufficientAccessRightSuggestion = `
Issue does not exist or you do not have permission to see it. This could mean:
1. The Jira ticket doesn't exist
2. You don't have permission to view it
3. Your JIRA_API_TOKEN is invalid or expired

Please verify:
- The Jira ticket URL is correct
- Your JIRA_API_TOKEN environment variable is set and valid
- You have access permissions for this ticket in Jira
`
)
