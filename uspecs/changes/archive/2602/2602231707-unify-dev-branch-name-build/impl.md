# Implementation plan: Unify dev branch name building

## Construction

### Core type changes

- [x] update: [internal/issue/issue.go](../../../../../internal/issue/issue.go)
  - remove: `Number` field from `IssueInfo`, rename `URL` to `Text`
  - add: Doc comment to `ID` field (GitHub: issue number, Jira: ticket key like `AIR-XXXX`, FreeForm: empty)
  - remove: `strconv.Atoi` conversion and `Number` assignment in `ParseIssueFromArgs`
  - remove: `wd` parameter from `ParseIssueFromArgs`
  - add: `BuildDevBranchName(info IssueInfo) (string, []string, error)` that dispatches based on `IssueInfo.Type`
  - add: `titleToKebab(id, title string)` helper
  - add: `fetchGithubIssueTitle(info IssueInfo)` helper (moved from `gitcmds`)

### Signature changes

- [x] update: [gitcmds/github.go](../../../../../gitcmds/github.go)
  - update: `LinkBranchToGithubIssue` parameter `issueNumber` from `int` to `string`
  - remove: `strconv.Itoa(issueNumber)` conversion inside `LinkBranchToGithubIssue`
  - remove: `BuildDevBranchName` function (logic moves to `issue.BuildDevBranchName`)

- [x] update: [internal/jira/jira.go](../../../../../internal/jira/jira.go)
  - remove: `GetJiraBranchName` function (logic moves to `issue.BuildDevBranchName`)
  - rename: `GetJiraIssueName` → `GetJiraIssueTitle`

### Caller updates

- [x] update: [internal/commands/dev.go](../../../../../internal/commands/dev.go)
  - update: Replace three-way `switch` with single `issue.BuildDevBranchName` call
  - update: `LinkBranchToGithubIssue` call to pass `issueInfo.ID` (string) instead of `issueInfo.Number` (int)
  - update: `ParseIssueFromArgs` call without `wd` parameter
  - update: Use `issueInfo.Text` instead of `issueInfo.URL`

- [x] update: [gitcmds/gitcmds.go](../../../../../gitcmds/gitcmds.go)
  - update: `GetJiraIssueName` → `GetJiraIssueTitle` call in `GetIssueDescription`

### Removed files

- [x] delete: [utils/branch.go](../../../../../utils/branch.go)
  - remove: `GetBranchName`, `GetTaskIDFromURL`, `splitQuotedArgs`, `clearEmptyArgs` (all unused after refactoring)

### Tests

- [x] update: [internal/issue/issue_test.go](../../../../../internal/issue/issue_test.go)
  - remove: `wantNum` field and `Number` assertions
  - update: `ParseIssueFromArgs` calls without `wd` parameter
  - add: `TestBuildDevBranchName` with 10 FreeForm test cases (relocated from `TestGetBranchName`)

- [x] delete: [utils/branch_test.go](../../../../../utils/branch_test.go)
  - remove: `TestGetBranchName` (relocated to `issue_test.go` as `TestBuildDevBranchName`)
  - remove: `TestGeRepoNameFromURL` (referenced non-existent `GetTaskIDFromURL`)

- [x] review: Verify no regressions
  - `go build ./...` compiles without errors
  - `go test ./...` — all tests pass

