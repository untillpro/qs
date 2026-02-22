package gitcmds

const (
	msgOkSeeYou                 = "Ok, see you"
	errMsgModFiles              = "you have modified files. Please first commit & push them"
	maximumBranchNameLength     = 100
	errMsgFailedToGetMainBranch = "failed to get main branch: %w"
	countOfZerosIn1000          = 3
	decimalBase                 = 10

	bitSizeOfInt64        = 64
	LargeFileHookFilename = "large-file-hook.sh"

	// Error message fragments for main branch sync issues (exported for test use)
	MsgCannotFastForward          = "Error: Cannot fast-forward merge upstream/%s into %s."
	MsgMainBranchDiverged         = "This usually means your local main branch has diverged from upstream."
	MsgConflictDetected           = "A conflict is detected in %s branch. To resolve the conflict, you CAN run the following commands to reset your %s branch to match upstream/%s and force-push the changes to your fork:"
	MsgToFixRunCommands           = "To fix this, run the following commands:"
	MsgGitCheckoutMain            = "git checkout main"
	MsgGitFetchUpstream           = "git fetch upstream"
	MsgGitResetHardUpstream       = "git reset --hard upstream/main"
	MsgGitPushOriginMainForce     = "git push origin main --force"
	MsgWarningOverwriteMainBranch = "Warning: This will overwrite your main branch on origin with the state of upstream/main, discarding any local or remote changes that diverge from upstream. Make sure you have backed up any important work before proceeding."
)

const largeFileHookContent = `
#!/bin/bash
totalsize=0
totalcnt=0

# Use -z to get NUL-separated records: "XY<space>path\0"
while IFS= read -r -d '' entry; do
  status="${entry:0:2}"         # e.g., "??", "A ", "AM", etc.
  path="${entry:3}"             # skip "XY " to get the path

  # consider untracked (??) and added (A*) files
  if [[ "$status" == "??" || "$status" == A* ]]; then
    extension="${path##*.}"
    if [[ "$extension" != "wasm" ]]; then
      # wc -c < file gives just the count; quote the path
      fs=$(wc -c < "$path")
      totalsize=$((totalsize + fs))
      totalcnt=$((totalcnt + 1))
    fi
  fi
done < <(git status --porcelain -z)

if (( $totalsize > 100000 )); then
  echo " Attempt to commit too large files: Files size = $totalsize"
	 exit 1
fi

if (( $totalcnt > 200 )); then
  echo " Attempt to commit too much files: Files number = $totalcnt"
	 exit 1
fi
`

type PRState string

const (
	PRStateOpen   PRState = "open"
	PRStateMerged PRState = "merged"
)
