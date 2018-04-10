package vcs

import (
	"fmt"
	"log"
	"os/exec"

	u "github.com/untillpro/qg/utils"
)

type gitConf struct {
}

// NewVCSGit returns git IVCS implementation
func NewVCSGit() IVCS {
	return &gitConf{}
}

func (conf *gitConf) Status() {
	p, err := exec.LookPath("git")
	u.PanicIfError(err)

	log.Println(p)

	o, err := exec.Command(p, "remote", "-v").Output()
	u.PanicIfError(err)
	fmt.Printf(string(o))
}
func (conf *gitConf) Upload() {
	fmt.Println("Git upload")
}

func (conf *gitConf) Download() {
	fmt.Println("Git download")
}

func (conf *gitConf) Gui() {
	fmt.Println("Git gui")
}
