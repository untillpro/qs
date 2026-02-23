---
registered_at: 2026-02-22T20:44:12Z
change_id: 2602222044-reduce-getrootfolder-calls
baseline: 54f427c0ff88495738c2b33ea386f73b080bde9d
archived_at: 2026-02-23T02:41:38Z
---

# Change request: Reduce redundant GetRootFolder calls in hook.go

## Why

`GetRootFolder()` spawns a `git rev-parse --show-toplevel` subprocess each time it is called. In `gitcmds/hook.go` it is called 5 times, but several calls are redundant within the same call chain — the result is already available from a caller up the stack.

## What

Eliminate redundant `GetRootFolder()` calls in `gitcmds/hook.go` by passing the already-resolved root dir down the call chain:

- `fillPreCommitFile`: accept `rootDir` parameter instead of calling `GetRootFolder` (caller `SetLocalPreCommitHook` and `SetGlobalPreCommitHook` already have it or can obtain it once)
- `isLargeFileHookContentUpToDate` and `updateLargeFileHookContent`: accept `rootDir` parameter instead of calling `GetRootFolder`; `EnsureLargeFileHookUpToDate` calls `GetRootFolder` once and passes it to both
- `getLocalHookFolder`: accept `rootDir` parameter instead of calling `GetRootFolder` (caller `LocalPreCommitHookExist` obtains it)
- Net result: 5 calls reduced to 3 (one each in `SetLocalPreCommitHook`, `EnsureLargeFileHookUpToDate`, `LocalPreCommitHookExist`)
