/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package gitcmds

import (
	"errors"
	"fmt"
	"strings"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
)

func AddNotes(wd string, notes []string) error {
	var filtered []string
	for _, s := range notes {
		if str := strings.TrimSpace(s); len(str) > 0 {
			filtered = append(filtered, str)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "notes", "append", "-m", strings.Join(filtered, caret+caret)). // double caret for backward compatibility
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)
		if len(stderr) > 0 {
			return errors.New(stderr)
		}
		return fmt.Errorf("failed to add note: %w", err)
	}
	printLn(stdout)
	return nil
}

// GetNotes returns notes for a branch
// Returns:
// - notes
// - revision count
// - error if any
func GetNotes(wd, branchName string) (notes []string, revCount int, err error) {
	mainBranchName, err := GetMainBranch(wd)
	if err != nil {
		return nil, 0, fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	return getNotesWithMainBranch(wd, branchName, mainBranchName)
}

func getNotesWithMainBranch(wd, branchName, mainBranchName string) (notes []string, revCount int, err error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "rev-list", mainBranchName+".."+branchName).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		return notes, 0, fmt.Errorf("failed to get commit list: %w", err)
	}
	if len(stdout) == 0 {
		return notes, 0, errors.New("error: No commits found in current branch")
	}

	revList := strings.Split(strings.TrimSpace(stdout), caret)
	for _, rev := range revList {
		stdout, stderr, err := new(exec.PipedExec).
			Command(git, "notes", "show", rev).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			if strings.Contains(stderr, "no note found") {
				continue
			}
			logger.Verbose(stderr)

			return notes, len(revList), fmt.Errorf("failed to get notes: %w", err)
		}
		rawNotes := strings.Split(stdout, caret)
		for _, rawNote := range rawNotes {
			note := strings.TrimSpace(rawNote)
			if len(note) > 0 {
				notes = append(notes, note)
			}
		}
	}

	if len(notes) == 0 {
		return notes, len(revList), errors.New("error: No notes found in current branch")
	}

	return notes, len(revList), nil
}

func GetBodyFromNotes(notes []string) string {
	b := ""
	if (len(notes) > 1) && strings.Contains(strings.ToLower(notes[0]), strings.ToLower(IssuePRTtilePrefix)) {
		for i, note := range notes {
			note = strings.TrimSpace(note)
			if (strings.Contains(note, "https://") && !strings.Contains(note, "/issues/")) || !strings.Contains(note, "https://") {
				strings.Split(strings.ReplaceAll(note, "\r\n", caret), "")
				if i > 0 && len(note) > 0 {
					b += note
				}
			}
		}
	}
	return b
}
