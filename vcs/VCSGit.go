package vcs

import (
	"fmt"
	"os"

	u "github.com/untillpro/qg/utils"
	"github.com/untillpro/qg/cmdupload"
)

type gitConf struct {
}

// NewVCSGit returns git IVCS implementation
func NewVCSGit() IVCS {
	return &gitConf{}
}

func (conf *gitConf) Status() {
	new(u.PipedExec).
		Command("git", "remote", "-v").
		Command("grep", "fetch").
		Run(os.Stdout, os.Stdout)

	new(u.PipedExec).
		Command("git", "status", "-s", "-b", "-uall").
		Run(os.Stdout, os.Stdout)

}
func (conf *gitConf) Upload() {
	fmt.Println("Git upload", uploadCmdMessage)
}

func (conf *gitConf) Download() {
	fmt.Println("Git download")
}

func (conf *gitConf) Gui() {
	fmt.Println("Git gui")
}
