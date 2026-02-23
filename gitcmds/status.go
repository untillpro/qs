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
	"strconv"
	"strings"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
)

// Status shows git repo status
func Status(wd string) error {
	stdout, stderr, err := new(exec.PipedExec).
		Command("git", "remote", "-v").
		WorkingDir(wd).
		Command("grep", fetch).
		Command("sed", "s/(fetch)//").
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return err
	}

	// Print the colorized git status output
	printLn(stdout)

	// Get git status output with colors for display
	statusStdout, statusStderr, err := new(exec.PipedExec).
		Command("git", "-c", "color.status=always", "status", "-s", "-b", "-uall").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(statusStderr)

		if len(statusStderr) > 0 {
			return errors.New(statusStderr)
		}

		return fmt.Errorf("git status failed: %w", err)
	}

	// Print the colorized git status output
	printLn(statusStdout)

	// Get clean output for parsing (without color codes)
	cleanStatusStdout, stderr, err := new(exec.PipedExec).
		Command("git", "status", "-s", "-b", "-uall").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to get clean status for parsing: %w", err)
	}

	files, err := getListOfChangedFiles(wd, cleanStatusStdout)
	if err != nil {
		return fmt.Errorf("failed to get list of changed and new files: %w", err)
	}

	// Calculate and display file size information using clean output
	if err := displaySummary(files); err != nil {
		logger.Verbose(fmt.Sprintf("Failed to calculate file sizes: %v", err))
	}

	return nil
}

// getListOfChangedFiles parses the git status output and returns lists of changed files
func getListOfChangedFiles(wd, statusOutput string) ([]fileInfo, error) {
	lines := strings.Split(strings.TrimSpace(statusOutput), "\n")
	if len(lines) == 0 {
		return []fileInfo{}, nil
	}

	files := make([]fileInfo, 0, len(lines))

	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "rev-parse", "--absolute-git-dir").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return nil, errors.New(stderr)
		}

		return nil, fmt.Errorf("failed to get absolute git dir: %w", err)
	}
	gitDir := strings.TrimSpace(stdout)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip a branch information line (starts with ##)
		if strings.HasPrefix(line, "##") {
			continue
		}

		// Parse git status line format: "XY filename"
		// nolint:revive
		if len(line) < 4 {
			continue // Line too short to contain status + space + filename
		}

		statusCode := strings.TrimSpace(line[:2])
		name := strings.TrimSpace(line[2:])

		// Unquote Git filenames (Git quotes filenames with spaces and special characters)
		name = unquoteGitFilename(name)

		oldName := name
		if name == "" {
			continue // Skip empty filenames
		}

		if strings.Contains(name, " -> ") {
			// Handle renamed files (format: "old -> new")
			parts := strings.Split(name, " -> ")
			if len(parts) == 2 {
				name = strings.TrimSpace(unquoteGitFilename(parts[1]))    // Use the new filename
				oldName = strings.TrimSpace(unquoteGitFilename(parts[0])) // Use the old filename
			} else {
				// TODO: Handle file name which -> as part of the file name
				continue // Skip malformed renamed files
			}
		}

		var (
			err1 error
			err2 error
		)

		oldSize := int64(0)
		newFileSize := int64(0)
		switch statusCode {
		case `A`, `AM`:
			newFileSize, err1 = getFileSize(wd, name)
		case `M`, `MM`, `RM`:
			newFileSize, err1 = getFileSize(wd, name)
			oldSize, err2 = getFileSizeFromHEAD(wd, gitDir, oldName)
		case `D`, `MD`:
			oldSize, err2 = getFileSizeFromHEAD(wd, gitDir, oldName)
		case `R`:
			newFileSize, err1 = getFileSize(wd, name)
			oldSize = newFileSize
		case `??`:
			newFileSize, err2 = getFileSize(wd, name)
		default:
			return nil, fmt.Errorf("unknown file status %s for file %s", statusCode, name)
		}

		if err2 != nil {
			return nil, err2
		}
		if err1 != nil {
			return nil, err1
		}

		sizeIncrease := newFileSize - oldSize
		if sizeIncrease < 0 {
			sizeIncrease = 0 // Ensure size increase is not negative
		}

		files = append(files, fileInfo{
			name:         name,
			sizeIncrease: sizeIncrease,
		})
	}

	return files, nil
}

func getFileSizeFromHEAD(wd, gitDir, fileName string) (int64, error) {
	repoRootDir := filepath.Dir(gitDir)
	// compute relative path
	relativePath, err := filepath.Rel(repoRootDir, wd)
	if err != nil {
		return 0, fmt.Errorf("failed to compute relative path from repo root to working dir: %w", err)
	}
	// workaround for Windows paths
	filePath := strings.ReplaceAll(filepath.Join(relativePath, fileName), "\\", "/")
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "cat-file", "-s", fmt.Sprintf("HEAD:%s", filePath)).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Error(stderr)

		if len(stderr) > 0 {
			return 0, errors.New(stderr)
		}

		return 0, fmt.Errorf("failed to get file size from HEAD for %s: %w", fileName, err)
	}

	return strconv.ParseInt(strings.TrimSpace(stdout), decimalBase, bitSizeOfInt64)
}

func getFileSize(wd, fileName string) (int64, error) {
	fileInfo, err := os.Stat(filepath.Join(wd, fileName))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("file %s does not exist: %w", fileName, err)
		}
		return 0, fmt.Errorf("failed to get file info for %s: %w", fileName, err)
	}

	return fileInfo.Size(), nil
}

// displaySummary calculates and displays file size information
func displaySummary(files []fileInfo) error {
	var (
		totalSize   int64
		largestFile fileInfo
	)

	for _, file := range files {
		totalSize += file.sizeIncrease
		if file.sizeIncrease > largestFile.sizeIncrease {
			largestFile = file
		}
	}

	// Print summary only if there are files to summarize
	if totalSize > 0 {
		fmt.Println()
		fmt.Println("Summary:")

		// Format total size with underscores and color
		totalSizeStr := formatSizeWithUnderscores(totalSize)
		fmt.Printf("  %s %s bytes\n", "Total positive delta:    ", totalSizeStr)

		// Find the largest file
		if largestFile.name != "" {
			largestSizeStr := formatSizeWithUnderscores(largestFile.sizeIncrease)
			fmt.Printf("  %s %s (%s bytes)\n", "Largest positive delta:  ", largestFile.name, largestSizeStr)
		}
	}

	return nil
}

func unquoteGitFilename(filename string) string {
	// Unquote Git filenames that are quoted with double quotes
	if strings.HasPrefix(filename, `"`) && strings.HasSuffix(filename, `"`) {
		filename = filename[1 : len(filename)-1]
		// Unescape common escape sequences
		filename = strings.ReplaceAll(filename, `\"`, `"`)
		filename = strings.ReplaceAll(filename, `\\`, `\`)
	}

	return filename
}

// formatSizeWithUnderscores formats a number with underscores as thousand separators
func formatSizeWithUnderscores(size int64) string {
	str := strconv.FormatInt(size, decimalBase)
	if len(str) <= countOfZerosIn1000 {
		return str
	}

	var result strings.Builder
	for i, digit := range str {
		if i > 0 && (len(str)-i)%countOfZerosIn1000 == 0 {
			result.WriteRune('_')
		}
		result.WriteRune(digit)
	}

	return result.String()
}
