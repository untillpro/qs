Quick source management tool

# Installation Prerequisites

## gh
- https://github.com/cli/cli
- MSI: https://github.com/cli/cli/releases/tag/v2.3.0
- chocolatey: choco install gh
  - https://docs.chocolatey.org/en-us/choco/setup#install-with-cmd.exe
  - Also
    - choco install gh
    - choco install git
    - choco install golang

## git

- $Git\usr\bin must be in PATH
  - gives `yes`, `grep`, `sed` and other Unix utilities

# Installation

go install github.com/untillpro/qs@latest

For linux additionally:
  sudo apt install xclip


# Usage
Must be run in cloned github repo folder.
Path qs.exe should be added to system PATH env variable.

- `qs`: show status of current folder

- `qs d`                 : Download sources (git pull)
- `qs u -m [message]`    : Upload sources (git add + commit + push)
- `qs r`                 : Create release
- `qs g`                 : Shows Git GUI
- `qs -h`, `qs --help`   : Help for qs
- `qs -v`, `qs --verbose`: Verbose output

- `qs fork`  		         : Forks repo to user's account and creates upstream
- `qs dev repo-name`     : Make new dev branch with name repo-name
Repo-name can be copied as [Name and Permanent link] from Project Kaiser task. 
- `qs dev -d`         	 : Deletes branches in user's repository, which were accepted & mergred to parent repo
- `qs pr`                : Create pull request. Title is taken from name of topic in "qs dev" command
- `qs pr merge [PR URL]`  : Merge pull request. 

Note:
  - `qs u` takes comment from clipboard. If current branch is "main/master", 
           and message is empty or very short (<3 symbols), qs willask to enter message.
           If the message is too short, it shows error:   
                  ----  Too short comment not allowed! --- 

# Read version from go file
Create file `version.go` in src folder with body
````go
package main

var Major = 1
var Minor = 0
var Patch = 0
````
Now you can get version from `go` file

