Quick source management tool

# Installation Prerequisites

## gh
- https://github.com/cli/cli
- MSI: https://github.com/cli/cli/releases/tag/v2.0.0
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

go install github.com/untillpro/qs@v1.10.0


# Usage
Must be run in cloned github repo folder.
Path qs.exe should be added to system PATH env variable.

- `qs`: show status of current folder

- `qs d`                 : Download sources (git pull)
- `qs u -m [message]`    : Upload sources (git add + commit + push)
- `qs r`                 : Creates a release
- `qs g`                 : Shows Git GUI
- `qs -h`, `qs --help`   : Help for qs
- `qs -v`, `qs --verbose`: Verbose output

- `qs fork`  : Forks repo to user's account and creates upstream. 
- `qs dev repo-name`     : Makes new dev branch with name repo-name. 
Repo-name can be copied as [Name and Permanent link] from Project Kaiser task. 

# Read version from go file
Create file `version.go` in src folder with body
````go
package main

var Major = 1
var Minor = 0
var Patch = 0
````
Now you can get version from `go` file
