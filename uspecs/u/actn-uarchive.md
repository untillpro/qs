# Action: Archive change

## Overview

Archive a completed change request folder.

## Instructions

Rules:

- Always read `uspecs/u/concepts.md` and `uspecs/u/conf.md` before proceeding and follow the definitions and rules defined there

Parameters:

- Input
  - Active Change Folder path
- Output
  - Folder moved to `$changes_folder/archive`

Flow:

- Identify Active Change Folder to archive, if unclear, ask user to specify folder name
- Execute `bash uspecs/u/scripts/uspecs.sh change archive <change-folder-name>`
  - Example: `bash uspecs/u/scripts/uspecs.sh change archive 2602211523-check-cmd-availability`
- Analyze output, show to user and stop
