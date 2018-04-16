package utils

import (
	"os/exec"
)

// https://github.com/b4b4r07/go-pipe/blob/master/README.md

// CmdPipe allows to execute commands in pipe
type CmdPipe struct {
	cmds []*exec.Cmd
}

// New creates a new pipe
func New() *CmdPipe {
	return &CmdPipe{}
}

// Add adds a command
func (pipe *CmdPipe) Add(name string, arg ...string) *CmdPipe {
	cmd := exec.Command("go", "build")
	pipe.cmds = append(pipe.cmds, cmd)
	return pipe
}

// Wd sets working directory for the last command
func (pipe *CmdPipe) Wd(wd string) {
	cmd := pipe.cmds[len(pipe.cmds)-1]
	cmd.Dir = wd
}

// Start starts the pipe
func (pipe *CmdPipe) Start() {
	for idx, cmd := range pipe.cmds[0 : len(pipe.cmds)-2] {
		nextCmd := pipe.cmds[idx+1]
		nextCmd.Stdin, _ = cmd.StdoutPipe()
	}
}
