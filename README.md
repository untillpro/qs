Quick source management tool

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

Example: 
````
qs dev Implement IEvents, IRecords (BasicUsage)  https://dev.heeus.io/launchpad/#!12057
````
creates developer branch with name "12057-Implement-IEvents-IRecords-BasicUsage"


# Read version from go file
Create file `version.go` in src folder with body
````go
package main

var Major = 1
var Minor = 0
var Patch = 0
````
Now you can get version from `go` file
