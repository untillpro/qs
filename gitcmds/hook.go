/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package gitcmds

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
)

// SetLocalPreCommitHook - s.e.
func SetLocalPreCommitHook(wd string) error {
	// Turn off globa1 hooks
	err := new(exec.PipedExec).
		Command(git, "config", "--global", "--get", "core.hookspath").
		Run(os.Stdout, os.Stdout)
	if err == nil {
		_, stderr, err := new(exec.PipedExec).
			Command(git, "config", "--global", "--unset", "core.hookspath").
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("failed to unset global hooks path: %w", err)
		}
	}
	dir, err := GetRootFolder(wd)
	if err != nil {
		return err
	}
	PreCommitHooksDirPath := filepath.Join(dir, ".git/hooks")

	if err := os.MkdirAll(PreCommitHooksDirPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	PreCommitFilePath := filepath.Join(PreCommitHooksDirPath, "pre-commit")

	// Check if the file already exists
	f, err := createOrOpenFile(PreCommitFilePath)
	if err != nil {
		return err
	}
	_ = f.Close()

	if !largeFileHookExist(PreCommitFilePath) {
		return fillPreCommitFile(wd, PreCommitFilePath)
	}

	return nil
}

func createOrOpenFile(filepath string) (*os.File, error) {
	_, err := os.Stat(filepath)
	var f *os.File
	if os.IsNotExist(err) {
		// Create file pre-commit
		f, err = os.Create(filepath)
		if err != nil {
			return nil, err
		}

		_, err = f.WriteString("#!/bin/bash\n")
	} else {
		f, err = os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, bashFilePerm)
	}
	if err != nil {
		return nil, err
	}

	return f, nil
}

func fillPreCommitFile(wd, myFilePath string) error {
	fPreCommit, err := createOrOpenFile(myFilePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = fPreCommit.Close()
	}()

	dir, err := GetRootFolder(wd)
	if err != nil {
		return err
	}
	fName := "/.git/hooks/" + LargeFileHookFilename
	lfPath := dir + fName

	lf, err := os.Create(lfPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = lf.Close()
	}()

	if _, err := lf.WriteString(largeFileHookContent); err != nil {
		return fmt.Errorf("failed to write large file hook content: %w", err)
	}

	preCommitContentBuf := strings.Builder{}
	preCommitContentBuf.WriteString("#!/bin/bash\n")
	preCommitContentBuf.WriteString("\n#Here is large files commit prevent is added by [qs]\n")
	preCommitContentBuf.WriteString("bash " + lfPath + caret)
	if _, err := fPreCommit.WriteString(preCommitContentBuf.String()); err != nil {
		return fmt.Errorf("failed to write pre-commit hook content: %w", err)
	}

	return new(exec.PipedExec).Command("chmod", "+x", myFilePath).Run(os.Stdout, os.Stdout)
}

// isLargeFileHookContentUpToDate checks if the current large-file-hook.sh content matches the expected content
func isLargeFileHookContentUpToDate(wd string) (bool, error) {
	dir, err := GetRootFolder(wd)
	if err != nil {
		return false, err
	}

	hookPath := filepath.Join(dir, ".git", "hooks", LargeFileHookFilename)

	// Check if the file exists
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		return false, nil // File doesn't exist, so it's not up to date
	}

	// Read the current content
	currentContent, err := os.ReadFile(hookPath)
	if err != nil {
		return false, fmt.Errorf("failed to read large file hook: %w", err)
	}

	// Compare with expected content
	return string(currentContent) == largeFileHookContent, nil
}

// updateLargeFileHookContent updates the large-file-hook.sh file with the current content
func updateLargeFileHookContent(wd string) error {
	dir, err := GetRootFolder(wd)
	if err != nil {
		return err
	}

	hookPath := filepath.Join(dir, ".git", "hooks", LargeFileHookFilename)

	// Create or overwrite the hook file
	lf, err := os.Create(hookPath)
	if err != nil {
		return fmt.Errorf("failed to create large file hook: %w", err)
	}
	defer func() {
		_ = lf.Close()
	}()

	if _, err := lf.WriteString(largeFileHookContent); err != nil {
		return fmt.Errorf("failed to write large file hook content: %w", err)
	}

	return nil
}

// EnsureLargeFileHookUpToDate checks and updates the large file hook if needed
func EnsureLargeFileHookUpToDate(wd string) error {
	upToDate, err := isLargeFileHookContentUpToDate(wd)
	if err != nil {
		return err
	}

	if !upToDate {
		return updateLargeFileHookContent(wd)
	}

	return nil
}
