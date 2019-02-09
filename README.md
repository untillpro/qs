Quick source management tool

# Usage

- `qg`: show status of current folder
- `qg d`: download sources (git pull)
- `qg u -m [message]`: upload sources (git add + commit + push)

# Under the Hood

- `vcs/cmdconfigs.go` defines generic vcs command configs
- `git/gitcmds` implement git commands 
