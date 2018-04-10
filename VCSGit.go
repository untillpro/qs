package main

import (
	"fmt"
)

type gitConf struct {
}

func NewGit() IVCS {
	return gitConf{}
}

func (conf *gitConf) upload() {
	fmt.Println("Git upload")
}

func (conf *gitConf) download() {
	fmt.Println("Git download")
}

func (conf *gitConf) gui() {
	fmt.Println("Git gui")
}
