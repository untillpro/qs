---
registered_at: 2026-02-21T15:24:38Z
change_id: 2602211523-u-cmd-single-msg-string
baseline: 9918beb30385990b9c46f289f3e6d3ad5b1e44f5
archived_at: 2026-02-21T15:41:38Z
---

# Change request: Make commit message a single string in u command

## Why

The `u` command currently accepts commit message as a string slice (`[]string`) via `-m` flag, but a commit message is semantically a single string. Using a slice adds unnecessary complexity for both the caller and the implementation.

## What

Change the commit message parameter type from `[]string` to `string` across all layers:

- `vcs.CfgUpload.Message` field type from `[]string` to `string`
- Flag binding from `StringSliceVarP` to `StringVarP`
- All usages that join or iterate over the message slice
