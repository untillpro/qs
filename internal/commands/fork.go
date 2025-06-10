package commands

import (
	"fmt"

	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/git"
	"github.com/untillpro/qs/internal/commands/helper"
)

func Fork() error {
	globalConfig()

	if !helper.CheckGH() {
		return fmt.Errorf("GitHub CLI check failed")
	}

	if ok, err := notCommittedRefused(); ok || err != nil {
		return fmt.Errorf("git refused to commit")
	}

	repo, err := git.Fork()
	if err != nil {
		return err
	}

	if err := git.MakeUpstream(repo); err != nil {
		logger.Verbose("Failed to set upstream: %v", err)
	}

	if err := git.PopStashedFiles(); err != nil {
		logger.Verbose("Failed to pop stashed files: %v", err)
	}

	return nil
}

func notCommittedRefused() (bool, error) {
	s, fileExists, err := git.ChangedFilesExist()
	if !fileExists {
		return false, err
	}
	fmt.Println(confMsgModFiles1)
	fmt.Println("----   " + s)
	fmt.Print(confMsgModFiles2)
	var response string
	_, _ = fmt.Scanln(&response)

	return response != pushYes, err
}
