# Implementation plan

## Construction

### Investigation findings

The following redundant git command execution patterns were identified across command workflows:

| Workflow             | Redundancy              | Times called | Needed |
|----------------------|-------------------------|:------------:|:------:|
| `qs u`               | `GetCurrentBranchName`  |      3       |   1    |
| `qs u`               | `GetMainBranch`         |      2       |   1    |
| `qs u`               | `git status` variants   |      2       |   1    |
| `qs pr` (dev branch) | `GetMainBranch`         |      5       |   1    |
| `qs pr` (dev branch) | `GetNotes`              |      4       |   1    |
| `qs pr` (dev branch) | `GetCurrentBranchName`  |      2       |   1    |
| `qs pr` (dev branch) | `HasRemote("upstream")` |      2       |   1    |
| `qs pr` (dev branch) | fetch notes from origin |      3       |   1    |
| `qs dev`             | `GetMainBranch`         |      3       |   1    |
| `qs dev`             | `HasRemote("upstream")` |      3       |   1    |
| `qs d`               | `GetMainBranch`         |      3       |   1    |
| `qs d`               | `IamInMainBranch`       |      2       |   1    |
| `qs d`               | `GetCurrentBranchName`  |      2       |   1    |

### Core library changes

- [x] update: [gitcmds/gitcmds.go](../../../../../gitcmds/gitcmds.go)
  - refactor: `Download()` — eliminate second `IamInMainBranch` call and redundant `GetMainBranch` by reusing values from the first call
  - refactor: `Upload()` — accept current branch name as parameter instead of re-calling `GetCurrentBranchName`
  - refactor: `SyncMainBranch()` — accept pre-computed `mainBranch` and `upstreamExists` to avoid re-fetching inside
  - refactor: `CreateDevBranch()` — accept pre-computed `mainBranch` to avoid redundant `GetMainBranch` call
  - refactor: `GetNotes()` — split into public wrapper + internal `getNotesWithMainBranch()` to avoid `GetMainBranch` call
  - refactor: `GetBranchType()` — return current branch name alongside type to let callers reuse it

- [x] update: [gitcmds/pr.go](../../../../../gitcmds/pr.go)
  - refactor: `Pr()` — reuse branch name from `GetBranchType`, get `mainBranch` once, use `getNotesWithMainBranch` for both notes calls
  - refactor: `createPRBranch()` — accept pre-computed notes, revCount, upstream status, and main branch from `Pr()`

### Command layer changes

- [x] update: [internal/commands/u.go](../../../../../internal/commands/u.go)
  - refactor: `U()` — capture `currentBranch` from `IamInMainBranch` and pass to `Upload`
  - refactor: `setCommitMessage()` — updated `GetBranchType` call to match new 3-return signature

- [x] update: [internal/commands/dev.go](../../../../../internal/commands/dev.go)
  - refactor: `Dev()` — replaced `IamInMainBranch` with `GetCurrentBranchName` + `GetMainBranch`, pass pre-computed values to `SyncMainBranch` and `CreateDevBranch`

### Tests

- [x] update: [gitcmds/gitcmds_test.go](../../../../../gitcmds/gitcmds_test.go)
  - no changes needed: tests use `GetNoteAndURL` helper, not changed signatures

- [x] update: [sys_test.go](../../../../../sys_test.go)
  - no changes needed: system tests use CLI framework, not direct function calls

- [x] review: Verify no regressions
  - `go build ./...` compiles without errors
  - `go test ./gitcmds/... ./internal/... -count=1 -short` — all tests pass

