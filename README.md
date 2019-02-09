Quick source management tool

# Usage

- `gs`: show status of current folder
- `gs d`: download sources (git pull)
- `gs u -m [message]`: upload sources (git add + commit + push)

# Under the Hood

- `vcs/cmdconfigs.go` defines generic vcs command configs
- `git/gitcmds` implement git commands 
