package utils

import (
	"io"
	"os/exec"
)

// https://github.com/b4b4r07/go-pipe/blob/master/README.md

// PipedExec allows to execute commands in pipe
type PipedExec struct {
	cmds []*pipedCmd
}

// Stderr redirection
const (
	StderrRedirectNone = iota
	StderrRedirectStdout
	StderrRedirectNull
)

type pipedCmd struct {
	stderrRedirection int
	cmd               *exec.Cmd
}

func (pExec *PipedExec) command(name string, stderrRedirection int, args ...string) *PipedExec {
	cmd := exec.Command(name, args...)
	lastIdx := len(pExec.cmds) - 1
	if lastIdx > -1 {
		var err error
		cmd.Stdin, err = pExec.cmds[lastIdx].cmd.StdoutPipe()
		PanicIfError(err)
	}
	pExec.cmds = append(pExec.cmds, &pipedCmd{stderrRedirection, cmd})
	return pExec
}

// Command adds a command to a pipe
func (pExec *PipedExec) Command(name string, args ...string) *PipedExec {
	return pExec.command(name, StderrRedirectNone, args...)
}

// Wd sets working directory for the last command
func (pExec *PipedExec) Wd(wd string) *PipedExec {
	pipedCmd := pExec.cmds[len(pExec.cmds)-1]
	pipedCmd.cmd.Dir = wd
	return pExec
}

// Run starts the pipe
func (pExec *PipedExec) Run(out io.Writer, err io.Writer) {
	for _, cmd := range pExec.cmds {
		if cmd.stderrRedirection == StderrRedirectNone {
			cmd.cmd.Stderr = err
		}
	}
	lastIdx := len(pExec.cmds) - 1
	if lastIdx < 0 {
		return
	}
	pExec.cmds[lastIdx].cmd.Stdout = out

	for _, cmd := range pExec.cmds {
		err := cmd.cmd.Start()
		if nil != err {
			panic("Failed:" + cmd.cmd.Path)
		}
	}

	pExec.cmds[lastIdx].cmd.Wait()
}
