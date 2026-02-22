# Implementation plan: Join Unstash and PopStashedFiles into single function

## Construction

- [x] update: [gitcmds/gitcmds.go](../../../../../gitcmds/gitcmds.go)
  - replace: `Unstash()` body with `PopStashedFiles()` logic (check `stashEntriesExist` first, then pop)
  - remove: `PopStashedFiles()` function
  - remove: `stashEntriesExist()` helper (inline the check into `Unstash`)
- [x] update: [internal/commands/fork.go](../../../../../internal/commands/fork.go)
  - replace: `gitcmds.PopStashedFiles(wd)` with `gitcmds.Unstash(wd)`

### Tests

- [x] review: `go build ./...` compiles without errors
- [x] review: `go vet ./...` passes without errors

