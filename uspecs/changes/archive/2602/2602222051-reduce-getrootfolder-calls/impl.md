# Implementation plan: Reduce redundant GetRootFolder calls in hook.go

## Construction

- [x] update: [gitcmds/hook.go](../../../../../gitcmds/hook.go)
  - refactor: `fillPreCommitFile` — replace `wd` param with `rootDir`, remove internal `GetRootFolder` call
  - refactor: `isLargeFileHookContentUpToDate` — replace `wd` param with `rootDir`, remove internal `GetRootFolder` call
  - refactor: `updateLargeFileHookContent` — replace `wd` param with `rootDir`, remove internal `GetRootFolder` call
  - refactor: `getLocalHookFolder` — replace `wd` param with `rootDir`, remove internal `GetRootFolder` call
  - refactor: `EnsureLargeFileHookUpToDate` — call `GetRootFolder` once, pass result to `isLargeFileHookContentUpToDate` and `updateLargeFileHookContent`
  - refactor: `SetLocalPreCommitHook` — pass existing `dir` to `fillPreCommitFile`
  - refactor: `SetGlobalPreCommitHook` — call `GetRootFolder` once, pass result to `fillPreCommitFile`
  - refactor: `LocalPreCommitHookExist` — call `GetRootFolder` once, pass result to `getLocalHookFolder`

### Tests

- [x] review: `go build ./...` compiles without errors
- [x] review: `go vet ./...` passes without errors

