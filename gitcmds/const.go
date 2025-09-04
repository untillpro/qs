package gitcmds

const (
	msgOkSeeYou                 = "Ok, see you"
	errMsgModFiles              = "you have modified files. Please first commit & push them"
	maximumBranchNameLength     = 100
	errMsgFailedToGetMainBranch = "failed to get main branch: %w"
	countOfZerosIn1000          = 3
	decimalBase                 = 10

	bitSizeOfInt64        = 64
	refsNotes             = "refs/notes/*:refs/notes/*"
	LargeFileHookFilename = "large-file-hook.sh"
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
