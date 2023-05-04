# Quick source management tool

## Installation Prerequisites

### gh    version > 2.27
- https://github.com/cli/cli
- MSI: https://github.com/cli/cli/releases/tag/v2.3.0
- chocolatey: choco install gh
  - https://docs.chocolatey.org/en-us/choco/setup#install-with-cmd.exe
  - Also
    - choco install gh
    - choco install git
    - choco install golang

### git

- $Git\usr\bin must be in PATH
  - gives `yes`, `grep`, `sed` and other Unix utilities

### xclip
For linux additionally:   sudo apt install xclip

## Installation

go install github.com/untillpro/qs@latest

## Basic Usage
Must be run in cloned github repo folder.
Path qs.exe should be added to system PATH env variable.

- `qs`: show status of current folder

- `qs d`                    : Download sources (git pull)
- `qs u -m [message]`       : Upload sources (git add + commit + push)
- `qs r`                    : Create release
- `qs g`                    : Shows Git GUI
- `qs -h`, `qs --help`      : Help for qs
- `qs -v`, `qs --verbose`   : Verbose output

- `qs fork`  		            : Forks repo to user's account and creates upstream
- `qs dev repo-name`        : Make new dev branch with name repo-name
                              Repo-name can be copied as 
                                - [Name and Permanent link] from Project Kaiser task. 
                                - URL of github issue with issue number.
                              If a buffer contains link on issue number, qs dev creates brnach and links it to github issue

- `qs dev -d`         	    : Deletes branches in user's repository, which were accepted & mergred to parent repo
- `qs dev -i`, 'qs dev --ignore-hook'    : Create developer branch and does not ask to add hook against large files.
- `qs pr`                   : Create pull request. Title is taken from name of topic in "qs dev" command
                            : If a branch is linked to github issue, qs pr makes a pull request, linked to that issue.
- `qs pr -d`, qs pr --draft : Create pull request draft.
- `qs pr merge [PR URL]`    : Merge pull request. 

Note:
  - `qs u` takes comment from clipboard. If current branch is "main/master", 
           and message is empty or very short (<3 symbols), qs willask to enter message.
           If the message is too short, it shows error:   
                  ----  Too short comment not allowed! --- 

## Prevent large commits

Command 'qs dev' creates a developer branch and after success, it shows the following question:

```
   Git pre-commit hook, preventing commit large files does not exist.
   Do you want to set hook(y/n)?
```

- On 'y', qs creates github local pre-commit hook script for current repository.

If local pre-commit hook found, 'qs dev' does not asks to create the hook.


