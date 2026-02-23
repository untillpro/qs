# Implementation plan: Deduplicate IamInMainBranch usage

## Construction

### Function rename and signature change

- [x] update: [gitcmds/gitcmds.go](../../../../../gitcmds/gitcmds.go)
  - rename: `IamInMainBranch` → `GetCurrentBranchInfo`
  - refactor: add `mainBranch` to return signature: `(currentBranch, mainBranch string, isMain bool, err error)`

### Callers using duplicated inline pattern

- [x] update: [gitcmds/download.go](../../../../../gitcmds/download.go)
  - refactor: `Download()` — replace inline `GetCurrentBranchName` + `GetMainBranch` + `EqualFold` with `GetCurrentBranchInfo` call

- [x] update: [internal/commands/dev.go](../../../../../internal/commands/dev.go)
  - refactor: `Dev()` — replace inline `GetCurrentBranchName` + `GetMainBranch` + `EqualFold` with `GetCurrentBranchInfo` call

### Existing callers

- [x] update: [internal/commands/u.go](../../../../../internal/commands/u.go)
  - refactor: `U()` — replace `IamInMainBranch` with `GetCurrentBranchInfo`, match new 4-return signature

### Verification

- [x] review: `go build ./...` compiles without errors
- [x] review: `go test ./gitcmds/... ./internal/... -count=1 -short` — all tests pass

