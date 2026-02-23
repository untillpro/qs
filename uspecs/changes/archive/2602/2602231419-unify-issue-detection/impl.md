# Implementation plan: Unify issue detection into single function

## Construction

### New shared types and function

- [x] create: [internal/issue/issue.go](../../../../../internal/issue/issue.go)
  - add: `IssueType` enum (`FreeForm`, `GitHub`, `Jira`)
  - add: `IssueInfo` struct with fields `Type IssueType`, `URL string`, `ID string`, `Number int` (GitHub issue number)
  - add: `ParseIssueFromArgs(wd string, args ...string) (IssueInfo, error)` that checks GitHub URL (contains `/issues/`, validates via `gh issue view`), then Jira URL (delegates to `jira.GetJiraTicketIDFromArgs`), returns typed result

### Caller updates

- [x] update: [internal/commands/dev.go](../../../../../internal/commands/dev.go)
  - remove: `argContainsGithubIssueLink` and `checkIssueLink` functions
  - update: replace two-call detection (`argContainsGithubIssueLink` + `jira.GetJiraTicketIDFromArgs`) with single `issue.ParseIssueFromArgs` call
  - update: simplify switch to use `issue.GitHub`, `issue.Jira`, `issue.FreeForm`
  - remove: unused imports (`strconv`)

### Tests

- [x] create: [internal/issue/issue_test.go](../../../../../internal/issue/issue_test.go)
  - add: table-driven tests for `ParseIssueFromArgs` covering GitHub URL, Jira URL, plain text, empty args

### Review

- [x] review
  - `go build ./...` compiles without errors
  - `go vet ./...` passes
  - existing tests in `internal/jira/` still pass (GetJiraTicketIDFromArgs unchanged)

