# Implementation plan: Support description with URL in qs dev input

## Construction

### Core parsing logic

- [x] update: [internal/issue/issue.go](../../../internal/issue/issue.go)
  - add: `extractURLFromText(text string) (url string, remainingText string)` - extracts first URL from text using `https?://[^\s]+` regex
  - add: `URL` and `Text` fields to `IssueInfo` struct; `URL` always holds the URL, `Text` always holds user description
  - add: `ExtractIDFromURL(rawURL string) string` - takes last slash-separated segment and trims leading `#` and `!`
  - update: `ParseIssueFromArgs` - extracts URL and text separately; validates text is non-empty for non-fetchable URLs
  - update: `BuildDevBranchName` - calls `notes.Serialize(info.URL, notes.BranchTypeDev, title)` for all issue types; stores only `notesObj` as comment (removed type-specific comment building)
  - update: `fetchGithubIssueTitle` - uses `info.URL` instead of `info.Text`
  - remove: `issuePRTitlePrefix` and `issueSign` constants (moved to `gitcmds` and no longer needed)

- [x] update: [internal/notes/notes.go](../../../internal/notes/notes.go)
  - add: `IssueURL` field to `Notes` struct - unified URL for all issue types (GitHub, Jira, non-fetchable)
  - deprecate: `GithubIssueURL` and `JiraTicketURL` fields - marked deprecated with `omitempty`; kept for deserialization of old notes only
  - update: `Serialize` signature - simplified from `(githubIssueURL, jiraTicketURL, branchType, description)` to `(issueURL, branchType, description)`
  - add: `ReadNotes(rawNotes []string) (*Notes, error)` - reads all formats: JSON and old plain-text (description + URL line scan)
  - update: `Deserialize` return signature - changed from `(*Notes, bool)` to `(*Notes, error)` with version validation via `supportedVersions` map

- [x] update: [gitcmds/gitcmds.go](../../../gitcmds/gitcmds.go)
  - update: `GetBranchType` - uses `ReadNotes` instead of `Deserialize`; removed `isOldStyledBranch` fallback
  - remove: `isOldStyledBranch` function and related constants (`httppref`, `oneSpace`, `IssuePRTtilePrefix`, `IssueSign`, `minRepoNameLength`)
  - update: `GetNoteAndURL` - uses `ReadNotes`; resolves URL from `IssueURL` then legacy fields with `nolint:staticcheck`
  - update: `GetIssueDescription` - local-first: reads `notesObj.Description` directly if present (no network fetch); falls back to API fetch for old notes lacking `Description`; adds `[id]` prefix for non-GitHub URLs via `ExtractIDFromURL`

### Tests

- [x] update: [internal/issue/issue_test.go](../../../internal/issue/issue_test.go)
  - add: `TestExtractURLFromText` covering same-line, newline-separated, URL-only, text-only, multiple URLs, empty input
  - add: `TestExtractIDFromURL` covering ProjectKaiser `#!` fragment, Heeus launchpad, Jira browse URL, GitHub issue URL, empty URL
  - update: `TestParseIssueFromArgs` - adds `wantURL` and `wantText` fields; covers all input format combinations
  - update: `TestBuildDevBranchName` - test cases use separate `Text` and `URL` fields; removed URL-in-branch-name cases

- [x] update: [internal/notes/notes_test.go](../../../internal/notes/notes_test.go)
  - update: `TestSerialize` - uses new single `issueURL` param; asserts `GithubIssueURL`/`JiraTicketURL` are empty in new notes
  - update: `TestDeserialize` - uses new error-return signature; adds unsupported version test case
  - add: `TestReadNotes` covering JSON format, old `Resolves issue` marker format, old plain-text + URL format, empty input

### Review

- [x] review
  - Run `go test -short ./internal/issue/... ./internal/notes/... ./gitcmds/...` - all pass
  - Run `go build ./...` - no compilation errors

## Quick start

Test with text + non-fetchable URL on same line:
```bash
# Copy to clipboard:
# TBlrBill.UpdateCustomerDisplays: does not display size modifiers https://dev.untill.com/projects/#!763090
qs dev
```

Test with text + non-fetchable URL on separate lines:
```bash
# Copy to clipboard (multi-line):
# TBlrBill.UpdateCustomerDisplays: does not display size modifiers
# https://dev.untill.com/projects/#!763090
qs dev
```

Test with fetchable URL only (existing behavior):
```bash
qs dev https://github.com/untillpro/qs/issues/123
```

Test with text only (existing behavior):
```bash
qs dev "Fix authentication bug"
```

Test with non-fetchable URL only (should error - text required):
```bash
# This should return an error
qs dev https://dev.untill.com/projects/#!763090
# Error: text is required when URL is non-fetchable
```
