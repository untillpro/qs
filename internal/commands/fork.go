package commands

import (
	"fmt"

	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/gitcmds"
	"github.com/untillpro/qs/internal/commands/helper"
)

func Fork(wd string) error {
	globalConfig()

	if !helper.CheckGH() {
		return fmt.Errorf("GitHub CLI check failed")
	}

	if ok, err := notCommittedRefused(wd); ok || err != nil {
		return fmt.Errorf("git refused to commit")
	}

	repo, err := gitcmds.Fork(wd)
	if err != nil {
		return err
	}

	if err := gitcmds.MakeUpstream(wd, repo); err != nil {
		logger.Verbose("Failed to set upstream: %v", err)
	}

	if err := gitcmds.PopStashedFiles(wd); err != nil {
		logger.Verbose("Failed to pop stashed files: %v", err)
	}

	return nil
}

func notCommittedRefused(wd string) (bool, error) {
	s, fileExists, err := gitcmds.ChangedFilesExist(wd)
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
