# Implementation plan: Fix dev branch creation flow

## Construction

### Remove redundant branch existence checks

- [x] update: [gitcmds/github.go](../../../../../gitcmds/github.go)
  - remove: Remote branch existence check (lines 64-81) from `CreateGithubLinkToIssue` — it uses uninitialized `branch` variable (bug) and is redundant since `Dev()` already checks existence

- [x] update: [gitcmds/dev.go](../../../../../gitcmds/dev.go)
  - remove: `checkRemoteBranchExistence` parameter from `CreateDevBranch` signature
  - remove: Remote branch existence check block (lines 48-64) inside `CreateDevBranch`

### Consolidate existence check in Dev()

- [x] update: [internal/commands/dev.go](../../../../../internal/commands/dev.go)
  - update: Replace local-only `branchExists` check with a combined local+remote existence check that works uniformly for both GitHub and Jira flows
  - update: Remove `checkRemoteBranchExistence` variable and pass updated signature to `CreateDevBranch`
  - update: Move branch existence check to happen after branch name is resolved but before user confirmation prompt, so the user is not asked to confirm creation of an already-existing branch

### Reorder GitHub branch creation and issue linking

- [x] update: [gitcmds/github.go](../../../../../gitcmds/github.go)
  - update: Renamed `CreateGithubLinkToIssue` to `LinkBranchToGithubIssue`, changed signature to return `(notes []string, err error)` instead of `(branch string, notes []string, err error)` since branch name is already known

- [x] update: [internal/commands/dev.go](../../../../../internal/commands/dev.go)
  - update: Reordered calls so `CreateDevBranch` runs first (creates local branch + pushes to origin), then `LinkBranchToGithubIssue` links the existing remote branch to the GitHub issue

### Unify dev branch name building

- [x] update: [gitcmds/github.go](../../../../../gitcmds/github.go) and [internal/commands/dev.go](../../../../../internal/commands/dev.go)
  - update: Changed `BuildDevBranchName` signature from `(string, error)` to `(string, []string, error)` — now returns notes (old-style comment, body, and JSON notes) alongside the branch name, matching the Jira/PK pattern
  - update: Removed note preparation from `LinkBranchToGithubIssue` (now returns `nil, nil` — linking only)
  - update: Unified `Dev()` branch resolution into a single `switch` dispatching to `BuildDevBranchName` / `GetJiraBranchName` / `GetBranchName`, all producing `(branch, notes, error)`

### Tests and review

- [x] review
  - `go build ./...` compiles without errors
  - `go vet ./...` passes

