# Implementation plan

## Construction

- [x] update: [vcs/cmdconfigs.go](../../../../../vcs/cmdconfigs.go)
  - update: `CfgUpload.Message` field type from `[]string` to `string`

- [x] update: [internal/cmdproc/cmdproc.go](../../../../../internal/cmdproc/cmdproc.go)
  - update: flag binding from `StringSliceVarP` to `StringVarP`, default from `[]string{}` to `""`

- [x] update: [internal/cmdproc/consts.go](../../../../../internal/cmdproc/consts.go)
  - update: `pushMsgComment` to reflect single string semantics

- [x] update: [internal/commands/u.go](../../../../../internal/commands/u.go)
  - refactor: `setCommitMessage()` — replace slice operations with single string length check, store `string` in context instead of `[]string`

- [x] update: [gitcmds/gitcmds.go](../../../../../gitcmds/gitcmds.go)
  - refactor: `Upload()` — read commit message as `string` from context, pass single `-m` flag instead of iterating over slice

- [x] update: [internal/systrun/types.go](../../../../../internal/systrun/types.go)
  - update: `ExpectationRemoteBranchWithCommitMessage()` — read commit message as `string` from context instead of `[]string`

