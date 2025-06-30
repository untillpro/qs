package gitcmds

const (
	pushFail       = "Ok, see you"
	errMsgModFiles = "You have modified files. Please first commit & push them."
)

const largeFileHookContent = `
#!/bin/bash
totalsize=0
totalcnt=0
readarray -t arr2 < <(git status --porcelain | awk '{if ($1 == "??" || $1 == "A") print $2}')
for row in "${arr2[@]}";do
  extension="${row##*.}"
  if [ "$extension" != "wasm" ]; then
    fs=$(wc -c $row | awk '{print $1}')
    totalsize=$(($totalsize+$fs))
    totalcnt=$(($totalcnt+1))
  fi
done
if (( $totalsize > 100000 )); then 
  echo " Attempt to commit too large files: Files size = $totalsize"
	 exit 1
fi
if (( $totalcnt > 200 )); then 
  echo " Attempt to commit too much files: Files number = $totalcnt"
	 exit 1
fi
`
