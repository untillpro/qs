package commands

import (
	"fmt"

	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/gitcmds"
)

func Fork(wd string) error {
	if ok, err := notCommittedRefused(wd); ok || err != nil {
		return fmt.Errorf("git refused to commit")
	}

	repo, err := gitcmds.Fork(wd)
	if err != nil {
		return err
	}

	if err := gitcmds.MakeUpstream(wd, repo); err != nil {
		logger.Verbose(fmt.Sprintf("Failed to set upstream: %v", err))
	}

	if err := gitcmds.PopStashedFiles(wd); err != nil {
		logger.Verbose(fmt.Sprintf("Failed to pop stashed files: %v", err))
	}

	return nil
}

func notCommittedRefused(wd string) (bool, error) {
	s, fileExists, err := gitcmds.ChangedFilesExist(wd)
	if !fileExists {
		return false, err
	}
	fmt.Println("You have modified files: ")
	fmt.Println("----   " + s)
	fmt.Print("All will be kept not commted. Continue(y/n)?")
	var response string
	_, _ = fmt.Scanln(&response)

	return response != pushYes, err
}
