---
registered_at: 2026-02-23T14:55:04Z
change_id: 2602231455-unify-dev-branch-name-build
baseline: 73e6cc2002b16ac67ead7c88671a34dc21e01413
archived_at: 2026-02-23T17:07:59Z
---

# Change request: Unify dev branch name building

## Why

Three separate functions (`BuildDevBranchName`, `GetJiraBranchName`, `GetBranchName` + manual `-dev` suffix) handle dev branch name construction depending on the issue type. This creates duplicated logic, scattered responsibilities, and the redundant `IssueInfo.Number` field (which is just `ID` converted to `int` and immediately converted back to `string` at the only call site).

## What

Combine `BuildDevBranchName`, `GetJiraBranchName`, and the freeform `GetBranchName`+`-dev` path into a single `BuildDevBranchName` function in the `issue` package that dispatches based on `IssueInfo.Type`:

- Move dev branch name building logic from `gitcmds.BuildDevBranchName` (GitHub), `jira.GetJiraBranchName` (Jira), and the freeform path in `dev.go` into `issue.BuildDevBranchName(info IssueInfo, args ...string)`
- Remove `IssueInfo.Number` field — use `ID` (string) only
- Add doc comment to `ID` field describing format per source type (GitHub: issue number, Jira: ticket key like `AIR-XXXX`, FreeForm: empty)
- Update `LinkBranchToGithubIssue` to accept `string` instead of `int` for issue number
- Remove now-unused `GetJiraBranchName` from `jira` package and `BuildDevBranchName` from `gitcmds` package
- Simplify `dev.go` caller to a single `issue.BuildDevBranchName` call
