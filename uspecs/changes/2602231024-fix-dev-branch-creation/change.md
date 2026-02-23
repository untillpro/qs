---
registered_at: 2026-02-23T10:24:58Z
change_id: 2602231024-fix-dev-branch-creation
baseline: b7c87527aa6501a58538c46ba20fcb43c9962e45
---

# Change request: Fix dev branch creation flow

## Why

The dev branch creation flow has redundant branch existence checks scattered across three separate functions (`buildDevBranchName` → `CreateGithubLinkToIssue` → `Dev()` → `CreateDevBranch`). The remote branch is checked in `CreateGithubLinkToIssue`, then local branch is checked in `Dev()`, then remote branch is checked again in `CreateDevBranch`. This makes the flow fragile, hard to follow, and inconsistent between GitHub and Jira issue workflows.

## What

Restructure the dev branch creation to have a single, clear pipeline for both GitHub and Jira issues:

- Remove the branch existence check from `CreateGithubLinkToIssue` (lines 64-81 of `gitcmds/github.go`) — it checks a variable `branch` that is empty at that point (bug), and the check is redundant anyway
- Remove the `checkRemoteBranchExistence` parameter from `CreateDevBranch` and the remote existence check inside it (lines 48-64 of `gitcmds/dev.go`) — existence is already validated before this function is called
- Consolidate branch existence checking into a single place in `Dev()` (`internal/commands/dev.go`) that works uniformly for both GitHub and Jira flows
- Separate concerns: branch name resolution (with notes preparation) happens first, then a single existence check, then branch creation/push
- Unify dev branch name building in `Dev()` — currently GitHub issues use `BuildDevBranchName` while Jira/PK issues use `GetJiraBranchName`/`GetBranchName` with separate `-dev` suffix logic; should be a single path that supports all issue types
- Move `CreateGithubLinkToIssue` to happen after `CreateDevBranch` — currently the GitHub issue is linked to the branch before the branch is actually created, which is the wrong order; the branch should be created first, then linked to the issue
