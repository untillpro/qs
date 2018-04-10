package vcs

import (
	"fmt"
)

type gitConf struct {
}

// NewVCSGit returns git IVCS implementation
func NewVCSGit() IVCS {
	return &gitConf{}
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
