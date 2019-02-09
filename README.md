Quick source management tool

# Usage

- `gg`: show status of current folder
- `gg d`: download sources (git pull)
- `gg u -m [message]`: upload sources (git add + commit + push)

# Under the Hood

- `vcs/cmdconfigs.go` defines generic vcs command configs
- `git/gitcmds` implement git commands 
