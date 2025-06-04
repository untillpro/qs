package commands

import (
	"fmt"
	"os"

	"github.com/untillpro/qs/git"
	"github.com/untillpro/qs/internal/commands/helper"
)

func Fork() {
	globalConfig()

	if !helper.CheckGH() {
		return
	}

	if notCommitedRefused() {
		return
	}

	repo, err := git.Fork()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)

		return
	}
	git.MakeUpstream(repo)
	git.PopStashedFiles()
}

func notCommitedRefused() bool {
	s, fileExists := git.ChangedFilesExist()
	if !fileExists {
		return false
	}
	fmt.Println(confMsgModFiles1)
	fmt.Println("----   " + s)
	fmt.Print(confMsgModFiles2)
	var response string
	fmt.Scanln(&response)
	return response != pushYes
}
