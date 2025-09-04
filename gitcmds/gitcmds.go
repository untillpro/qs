package gitcmds

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	goGitPkg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	contextPkg "github.com/untillpro/qs/internal/context"
	"github.com/untillpro/qs/internal/helper"
	"github.com/untillpro/qs/internal/jira"
	notesPkg "github.com/untillpro/qs/internal/notes"
	"github.com/untillpro/qs/utils"
)

const (
	mimm              = "-m"
	slash             = "/"
	caret             = "\n"
	caretByte         = '\n'
	git               = "git"
	push              = "push"
	pull              = "pull"
	fetch             = "fetch"
	branch            = "branch"
	origin            = "origin"
	originSlash       = "origin/"
	httppref          = "https"
	pushYes           = "y"
	MsgPreCommitError = "Attempt to commit too"
	MsgCommitForNotes = "Commit for keeping notes in branch"
	oneSpace          = " "
	err128            = "128"

	repoNotFound            = "git repo name not found"
	userNotFound            = "git user name not found"
	ErrAlreadyForkedMsg     = "you are in fork already\nExecute 'qs dev [branch name]' to create dev branch"
	ErrMsgPRNotesImpossible = "pull request without comments is impossible"
	DefaultCommitMessage    = "wip"

	IssuePRTtilePrefix = "Resolves issue"
	IssueSign          = "Resolves #"

	minIssueNoteLength             = 10
	minPRTitleLength               = 8
	minRepoNameLength              = 4
	bashFilePerm       os.FileMode = 0644

	issuelineLength  = 5
	issuelinePosOrg  = 4
	issuelinePosRepo = 3
)

func CheckIfGitRepo(wd string) (string, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command("git", "status", "-s").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if strings.Contains(err.Error(), err128) {
			err = errors.New("this is not a git repository")
		} else if len(stderr) > 0 {
			err = errors.New(strings.TrimSpace(stderr))
		}
	}

	return stdout, err
}

// ChangedFilesExist s.e.
func ChangedFilesExist(wd string) (string, bool, error) {
	files, err := CheckIfGitRepo(wd)
	uncommitedFiles := strings.TrimSpace(files)

	return uncommitedFiles, len(uncommitedFiles) > 0, err
}

// stashEntriesExist checks if there are any stash entries in the git repository
func stashEntriesExist(wd string) (bool, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "stash", "list").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return false, errors.New(stderr)
		}

		return false, fmt.Errorf("failed to check stash entries: %w", err)
	}
	stashEntries := strings.TrimSpace(stdout)

	return len(stashEntries) > 0, nil
}

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

		if strings.Contains(err.Error(), err128) {
			return errors.New("this is not a git repository")
		}

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
		// Fallback to using the colored output for parsing
		cleanStatusStdout = statusStdout

		return fmt.Errorf("failed to get clean status for parsing: %w", err)
	}

	files, err := getListOfChangedFiles(wd, cleanStatusStdout)
	if err != nil {
		return fmt.Errorf("failed to get list of changed and new files: %w", err)
	}

	// Calculate and display file size information using clean output
	if err := displaySummary(wd, files); err != nil {
		logger.Verbose(fmt.Sprintf("Failed to calculate file sizes: %v", err))
	}

	return nil
}

// showDiffsOfChangedFiles shows diffs for modified files in the git repository
// nolint: unused
func showDiffsOfChangedFiles(wd string, files []FileInfo) error {
	if len(files) == 0 {
		fmt.Println("No changed files to show diffs for.")

		return nil
	}

	fmt.Println()
	fmt.Println("Diffs for changed files:")

	for _, file := range files {
		if file.status != fileStatusModified {
			continue // Only show diffs for modified files
		}

		stdout, stderr, err := new(exec.PipedExec).
			Command(git, "diff", "--color=always", file.name).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("failed to show diff for %s: %w", file.name, err)
		}
		printLn(stdout)
	}

	return nil
}

// getListOfChangedFiles parses the git status output and returns lists of changed files
func getListOfChangedFiles(wd, statusOutput string) ([]FileInfo, error) {
	lines := strings.Split(strings.TrimSpace(statusOutput), "\n")
	if len(lines) == 0 {
		return []FileInfo{}, nil
	}

	files := make([]FileInfo, 0, len(lines))

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
			status fileStatus
			err1   error
			err2   error
		)

		switch statusCode {
		case `A`, `AM`:
			status = fileStatusAdded
		case `M`, `MM`, `RM`:
			status = fileStatusModified
		case `D`:
			status = fileStatusDeleted
		case `R`:
			status = fileStatusRenamed
		case `??`:
			status = fileStatusUntracked
		default:
			return nil, fmt.Errorf("unknown file status %s for file %s", statusCode, name)
		}

		oldSize := int64(0)
		newFileSize := int64(0)
		switch status {
		case fileStatusAdded:
			newFileSize, err1 = getFileSize(wd, name)
		case fileStatusModified:
			newFileSize, err1 = getFileSize(wd, name)
			oldSize, err2 = getFileSizeFromHEAD(wd, gitDir, oldName)
		case fileStatusDeleted:
			oldSize, err2 = getFileSizeFromHEAD(wd, gitDir, oldName)
		case fileStatusRenamed:
			newFileSize, err1 = getFileSize(wd, name)
			oldSize = newFileSize
		case fileStatusUntracked:
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

		files = append(files, FileInfo{
			status:       status,
			name:         name,
			oldName:      oldName,
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

	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "cat-file", "-s", fmt.Sprintf("HEAD:%s", filepath.Join(relativePath, fileName))).
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
func displaySummary(wd string, files []FileInfo) error {
	var (
		totalSize   int64
		largestFile FileInfo
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

/*
	- Pull
	- Get current verson
	- If PreRelease is not empty fails
	- Calculate target version
	- Ask
	- Save version
	- Commit
	- Tag with target version
	- Bump current version
	- Commit
	- Push commits and tags
*/

// Release current branch. Remove PreRelease, tag, bump version, push
func Release(wd string) error {

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Pulling")
	stdout, stderr, err := new(exec.PipedExec).
		Command("git", pull).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("error pulling: %w", err)
	}
	logger.Verbose(stdout)

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Reading current version")
	currentVersion, err := utils.ReadVersion()
	if err != nil {
		return fmt.Errorf("error reading file 'version': %w", err)
	}
	if len(currentVersion.PreRelease) == 0 {
		return errors.New("pre-release part of version does not exist: " + currentVersion.String())
	}

	// Calculate target version

	targetVersion := currentVersion
	targetVersion.PreRelease = ""

	fmt.Printf("Version %v will be tagged, bumped and pushed, agree? [y]", targetVersion)
	var response string
	_, _ = fmt.Scanln(&response)
	if response != "y" {
		return errors.New("release aborted by user")
	}

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Updating 'version' file")
	if err := targetVersion.Save(); err != nil {
		return fmt.Errorf("error saving file 'version': %w", err)
	}

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Committing target version")
	{
		params := []string{"commit", "-a", mimm, "#scm-ver " + targetVersion.String()}
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, params...).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("error committing target version: %w", err)
		}
		logger.Verbose(stdout)
	}

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Tagging")
	{
		tagName := "v" + targetVersion.String()
		n := time.Now()
		params := []string{"tag", mimm, "Version " + tagName + " of " + n.Format("2006/01/02 15:04:05"), tagName}
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, params...).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("error tagging version: %w", err)
		}
		logger.Verbose(stdout)
	}

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Bumping version")
	newVersion := currentVersion
	{
		newVersion.Minor++
		newVersion.PreRelease = "SNAPSHOT"
		if err := targetVersion.Save(); err != nil {
			return fmt.Errorf("error saving file 'version': %w", err)
		}
	}

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Committing new version")
	{
		params := []string{"commit", "-a", mimm, "#scm-ver " + newVersion.String()}
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, params...).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("error committing new version: %w", err)
		}
	}

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Pushing to origin")
	{
		params := []string{push, "--follow-tags", origin}
		err = helper.Retry(func() error {
			stdout, stderr, err = new(exec.PipedExec).
				Command(git, params...).
				WorkingDir(wd).
				RunToStrings()
			if err != nil {
				if len(stderr) > 0 {
					return errors.New(stderr)
				}

				return fmt.Errorf("error pushing to origin: %w", err)
			}

			return nil
		})
		if err != nil {
			if len(stderr) > 0 {
				logger.Verbose(stderr)
			}

			return err
		}
		logger.Verbose(stdout)
	}

	return nil
}

// Upload uploads sources to git repo
func Upload(cmd *cobra.Command, wd string) error {
	commitMessageParts := cmd.Context().Value(contextPkg.CtxKeyCommitMessage).([]string)

	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "add", ".").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("git add failed: %w", err)
	}
	logger.Verbose(stdout)

	params := []string{"commit", "-a"}
	for _, m := range commitMessageParts {
		params = append(params, mimm, m)
	}

	_, stderr, err = new(exec.PipedExec).
		Command(git, params...).
		WorkingDir(wd).
		RunToStrings()
	if strings.Contains(stderr, MsgPreCommitError) {
		var response string
		fmt.Println("")
		printLn(strings.TrimSpace(stderr))
		fmt.Print("Do you want to commit anyway(y/n)?")
		_, _ = fmt.Scanln(&response)

		if response != "y" {
			return nil
		}

		params = append(params, "-n")
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, params...).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("git commit failed: %w", err)
		}
	}

	// make pull before push
	stdout, stderr, err = new(exec.PipedExec).
		Command(git, pull).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("error pulling before push: %w", err)
	}

	brName, err := GetCurrentBranchName(wd)
	if err != nil {
		return err
	}

	// Push notes to origin
	err = helper.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, push, origin, refsNotes).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("git push notes failed: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if helper.IsTest() {
		helper.Delay()
	}

	// Push branch to origin
	// Check if branch already has upstream tracking
	hasUpstream, err := hasUpstreamBranch(wd, brName)
	if err != nil {
		return fmt.Errorf("failed to check upstream branch: %w", err)
	}

	// Only use -u flag if upstream is not already configured
	pushArgs := []string{push, origin, brName}
	if !hasUpstream {
		pushArgs = []string{push, "-u", origin, brName}
	}

	err = helper.Retry(func() error {
		var pushErr error
		stdout, stderr, pushErr = new(exec.PipedExec).
			Command(git, pushArgs...).
			WorkingDir(wd).
			RunToStrings()
		if pushErr != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("git push failed: %w", pushErr)
		}

		return nil
	})
	if err != nil {
		return err
	}
	logger.Verbose(stdout)

	return nil
}

// Stash stashes uncommitted changes
func Stash(wd string) error {
	_, stderr, err := new(exec.PipedExec).
		Command("git", "stash").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("git stash failed: %w - %s", err, stderr)
	}

	return nil
}

// Unstash pops the latest stash
func Unstash(wd string) error {
	stdout, stderr, err := new(exec.PipedExec).
		Command("git", "stash", "pop").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		const msg = "No stash entries found"
		if strings.Contains(stdout, msg) || strings.Contains(stderr, msg) {
			return nil // No stash to pop, return nil
		}

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("git stash pop failed: %w", err)
	}

	return nil
}

func HaveUncommittedChanges(wd string) (bool, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "status", "--porcelain").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return false, errors.New(stderr)
		}

		return false, fmt.Errorf("failed to check if there are uncommitted changes: %w", err)
	}

	return len(stdout) > 0, nil
}

// Download sources from git repo
func Download(wd string) error {
	// Step 1: Exit if there are uncommitted changes
	uncommittedChanges, err := HaveUncommittedChanges(wd)
	if err != nil {
		return err
	}

	if uncommittedChanges {
		return errors.New("there are uncommitted changes in the repository")
	}

	var (
		stderr string
		stdout string
	)
	// Step 2: fetch origin --prune
	err = helper.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, fetch, origin, "--prune").
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("failed to fetch origin --prune: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}
	logger.Verbose(stdout)

	// Step 3: git fetch origin --force refs/notes/*:refs/notes/*
	err = helper.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, fetch, origin, "--force", refsNotes).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("failed to fetch notes: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}
	logger.Verbose(stdout)

	// Get current branch info
	currentBranchName, isMain, err := IamInMainBranch(wd)
	if err != nil {
		return err
	}

	// Get the name of the main branch
	mainBranchName, err := GetMainBranch(wd)
	if err != nil {
		return fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	// check out on the main branch
	if !isMain {
		if err := CheckoutOnBranch(wd, mainBranchName); err != nil {
			return err
		}
	}

	// Step 4: merge origin Main => Main
	_, stderr, err = new(exec.PipedExec).
		Command(git, "merge", fmt.Sprintf("origin/%s", mainBranchName)).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to merge origin/%s: %w", mainBranchName, err)
	}

	// check out back on the previous branch
	if !isMain {
		if err := CheckoutOnBranch(wd, currentBranchName); err != nil {
			return err
		}
	}

	// Step 5: If not on Main and the remote tracking branch exists merge local branch with the remote branch
	if !isMain {
		var hasRemoteBranch bool

		hasRemoteBranch, err = hasRemoteTrackingBranch(wd, currentBranchName)
		if err != nil {
			return fmt.Errorf("failed to check remote tracking branch: %w", err)
		}

		if hasRemoteBranch {
			_, stderr, err = new(exec.PipedExec).
				Command(git, "merge", fmt.Sprintf("origin/%s", currentBranchName)).
				WorkingDir(wd).
				RunToStrings()
			if err != nil {
				logger.Verbose(stderr)

				if len(stderr) > 0 {
					return errors.New(stderr)
				}

				return fmt.Errorf("failed to merge origin/%s: %w", currentBranchName, err)
			}
		}
	}

	// Step 6: If upstream exists - pull upstream/Main --no-rebase => Main
	upstreamExists, err := HasRemote(wd, "upstream")
	if err != nil {
		return fmt.Errorf("failed to check if upstream exists: %w", err)
	}

	if upstreamExists {
		err = helper.Retry(func() error {
			stdout, stderr, err = new(exec.PipedExec).
				Command(git, pull, "--no-rebase", "upstream", mainBranchName).
				WorkingDir(wd).
				RunToStrings()
			if err != nil {
				logger.Verbose(stderr)
				if len(stderr) > 0 {
					return errors.New(stderr)
				}

				return fmt.Errorf("failed to pull upstream/%s --no-rebase: %w", mainBranchName, err)
			}

			return nil
		})
		if err != nil {
			return err
		}
		logger.Verbose(stdout)
	}

	return nil
}

func CheckoutOnBranch(wd, branchName string) error {
	_, stderr, err := new(exec.PipedExec).
		Command(git, "checkout", branchName).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to checkout on %s: %w", branchName, err)
	}

	return nil
}

func GetBranchesWithRemoteTracking(wd, remoteName string) ([]string, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "branch", "-vv").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return nil, errors.New(stderr)
		}

		return nil, fmt.Errorf("failed to get list of rtBranchLines with remote tracking: %w", err)
	}
	// No remote tracking branches found
	if len(stdout) == 0 {
		return nil, nil
	}

	mainBranchName, err := GetMainBranch(wd)
	if err != nil {
		return nil, fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	// remote tracking branch suffixes:
	// - gone - branch is deleted on remote
	// - ahead 1 - branch is ahead of remote by 1 commit, could be ahead by merge commit
	rtBranchPattern := fmt.Sprintf(`\[%s/([A-Za-z0-9\-_\.\/]+):? ?(gone)?(ahead 1)?\]`, remoteName)
	re := regexp.MustCompile(rtBranchPattern)

	rtBranchLines := strings.Split(strings.TrimSpace(stdout), caret)
	branchesWithRemoteTracking := make([]string, 0, len(rtBranchLines))
	for _, rtBranchLine := range rtBranchLines {
		matches := re.FindStringSubmatch(rtBranchLine)
		if matches == nil {
			continue
		}

		branchName := matches[1]
		// exclude the main branch from the list
		if branchName == mainBranchName {
			continue
		}
		branchesWithRemoteTracking = append(branchesWithRemoteTracking, branchName)
	}

	return branchesWithRemoteTracking, nil
}

// Gui shows gui
func Gui(wd string) error {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "gui").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("git gui failed: %w", err)
	}
	printLn(stdout)

	return nil
}

func getFullRepoAndOrgName(wd string) (string, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "config", "--local", "remote.origin.url").
		WorkingDir(wd).
		Command("sed", "s/\\.git$//").
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", errors.New(stderr)
		}

		return "", fmt.Errorf("failed to get remote origin URL: %w", err)
	}

	return strings.TrimSuffix(strings.TrimSpace(stdout), slash), nil
}

// GetRepoAndOrgName - from .git/config
func GetRepoAndOrgName(wd string) (repo string, org string, err error) {
	repoURL, err := getFullRepoAndOrgName(wd)
	if err != nil {
		return "", "", err
	}

	org, repo, _, err = ParseGitRemoteURL(repoURL)
	if err != nil {
		return "", "", err
	}

	return
}

// ParseGitRemoteURL extracts account, repository name, and token from a git remote URL.
// Handles HTTPS format (https://github.com/account/repo.git),
// and HTTPS with token (https://account:token@github.com/account/repo.git or https://oauth2:token@github.com/repo.git)
func ParseGitRemoteURL(remoteURL string) (account, repo string, token string, err error) {
	if remoteURL == "" {
		return "", "", "", errors.New("remote URL is empty")
	}

	remoteURL = strings.TrimSuffix(remoteURL, ".git")
	// Handle HTTPS format: https://github.com/account/repo
	// or https://account:token@github.com/account/repo
	if strings.HasPrefix(remoteURL, "http") {
		u, err := url.Parse(remoteURL)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to parse URL: %w", err)
		}

		// Extract token if present in the userinfo section
		if u.User != nil {
			password, hasPassword := u.User.Password()
			if hasPassword {
				token = password
			}
		}

		// Remove leading '/' if any
		path := strings.TrimPrefix(u.Path, slash)
		pathParts := strings.Split(path, slash)

		if len(pathParts) < 2 {
			return "", "", "", fmt.Errorf("invalid repository path in URL: %s", remoteURL)
		}

		// If no token was found or username is oauth2 (common for token auth without real account name)
		// use the first path component as account
		if u.User != nil && u.User.Username() == "oauth2" {
			account = pathParts[0]
		} else if u.User != nil && u.User.Username() != "" {
			account = u.User.Username()
		} else {
			account = pathParts[0]
		}

		return account, pathParts[1], token, nil
	}

	return "", "", "", fmt.Errorf("unsupported git URL format: %s", remoteURL)
}

func IsMainOrg(wd string) (bool, error) {
	_, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return false, err
	}
	userName, err := getUserName(wd)

	return org != userName, err
}

// Fork repo
func Fork(wd string) (string, error) {
	repo, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return "", err
	}

	if len(repo) == 0 {
		return "", errors.New(repoNotFound)
	}

	remoteURL := GetRemoteUpstreamURL(wd)
	if len(remoteURL) > 0 {
		return repo, errors.New(ErrAlreadyForkedMsg)
	}

	if ok, err := IsMainOrg(wd); !ok || err != nil {
		if err != nil {
			return repo, fmt.Errorf("IsMainOrg error: %w", err)
		}

		return repo, errors.New(ErrAlreadyForkedMsg)
	}

	_, chExist, err := ChangedFilesExist(wd)
	if err != nil {
		return "", err
	}
	if chExist {
		stdout, stderr, err := new(exec.PipedExec).
			Command(git, "add", ".").
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return repo, errors.New(stderr)
			}

			return repo, fmt.Errorf("git add failed: %w", err)
		}
		printLn(stdout)

		stdout, stderr, err = new(exec.PipedExec).
			Command(git, "stash").
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return repo, errors.New(stderr)
			}

			return repo, fmt.Errorf("git stash failed: %w", err)
		}
		printLn(stdout)
	}

	var (
		stdout string
		stderr string
	)
	err = helper.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command("gh", "repo", "fork", org+slash+repo, "--clone=false").
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("failed to fork repository: %w", err)
		}

		return nil
	})
	if err != nil {
		return repo, err
	}
	printLn(stdout)

	// Get current user name to verify fork
	userName, err := getUserName(wd)
	if err != nil {
		logger.Verbose(fmt.Sprintf("Failed to get user name for verification: %v", err))

		return repo, err
	}

	// Verify fork was created and is accessible with retry
	err = helper.Retry(func() error {
		// Try to get user email to get a valid token context, then verify repo
		userEmail, emailErr := helper.GetUserEmail()
		if emailErr != nil {
			return fmt.Errorf("failed to verify GitHub authentication: %w", emailErr)
		}
		logger.Verbose(fmt.Sprintf("Verified GitHub authentication for user: %s", userEmail))

		// Verify the forked repository exists and is accessible
		return helper.VerifyGitHubRepoExists(userName, repo, "")
	})
	if err != nil {
		logger.Verbose(fmt.Sprintf("Fork verification failed: %v", err))

		return repo, fmt.Errorf("fork verification failed: %w", err)
	}
	_, _ = fmt.Fprintln(os.Stdout, "Fork created and verified successfully")

	return repo, nil
}

func GetRemoteUpstreamURL(wd string) string {
	stdout, _, err := new(exec.PipedExec).
		Command(git, "config", "--local", "remote.upstream.url").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(stdout)
}

func PopStashedFiles(wd string) error {
	if ok, err := stashEntriesExist(wd); !ok {
		return err
	}

	_, stderr, err := new(exec.PipedExec).
		Command(git, "stash", "pop").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("PopStashedFiles error: %s", stderr)
	}

	return nil
}

func GetMainBranch(wd string) (string, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, branch, "-r").
		WorkingDir(wd).
		Command("grep", "-E", "(/main|/master)([^a-zA-Z0-9]|$)").
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", errors.New(stderr)
		}

		return "", fmt.Errorf("failed to get main branch: %w", err)
	}
	logger.Verbose(stdout)

	// Check if the output contains "main" or "master"
	mainBranchFound := strings.Contains(stdout, "/main")
	masterBranchFound := strings.Contains(stdout, "/master")

	switch {
	case mainBranchFound && masterBranchFound:
		return "", fmt.Errorf("both main and master branches found")
	case mainBranchFound:
		return "main", nil
	case masterBranchFound:
		return "master", nil
	}

	return "", errors.New("neither main nor master branches found")
}

func getUserName(wd string) (string, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command("gh", "api", "user").
		WorkingDir(wd).
		Command("jq", "-r", ".login").
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", errors.New(stderr)
		}

		return "", fmt.Errorf("failed to get user name: %w", err)
	}

	return strings.TrimSpace(stdout), nil
}

func MakeUpstreamForBranch(wd string, parentRepo string) error {
	_, stderr, err := new(exec.PipedExec).
		Command(git, "remote", "add", "upstream", "https://github.com/"+parentRepo).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to add upstream remote: %w", err)
	}

	return nil
}

// MakeUpstream s.e.
func MakeUpstream(wd string, repo string) error {
	userName, err := getUserName(wd)
	if err != nil {
		return fmt.Errorf("failed to get user name: %w", err)
	}

	if len(userName) == 0 {
		return errors.New(userNotFound)
	}

	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "remote", "rename", "origin", "upstream").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to rename origin to upstream: %w", err)
	}
	printLn(stdout)

	stdout, stderr, err = new(exec.PipedExec).
		Command(git, "remote", "add", "origin", "https://github.com/"+userName+slash+repo).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to add origin remote: %w", err)
	}
	printLn(stdout)

	// delay to ensure remote is added
	if helper.IsTest() {
		helper.Delay()
	}

	err = helper.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, "fetch", "origin").
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("failed to fetch origin: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}
	logger.Verbose(stdout)

	stdout, stderr, err = new(exec.PipedExec).
		Command(git, branch, "--set-upstream-to", originSlash+mainBranch, mainBranch).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to set upstream for main branch: %w", err)
	}
	printLn(stdout)

	return nil
}

func GetIssueRepoFromURL(url string) (repoName string) {
	if len(url) < 2 {
		return
	}
	if strings.HasSuffix(url, slash) {
		url = url[:len(url)-1]
	}

	arr := strings.Split(url, slash)
	if len(arr) > issuelineLength {
		repo := arr[len(arr)-issuelinePosRepo]
		org := arr[len(arr)-issuelinePosOrg]
		repoName = org + slash + repo
	}

	return
}

// CreateGithubLinkToIssue create a link between an upstream GitHub issue and the dev branch
func CreateGithubLinkToIssue(wd, parentRepo, githubIssueURL string, issueNumber int, args ...string) (branch string, notes []string, err error) {
	repo, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return "", nil, fmt.Errorf("GetRepoAndOrgName failed: %w", err)
	}

	if len(repo) == 0 {
		return "", nil, errors.New(repoNotFound)
	}

	strIssueNum := strconv.Itoa(issueNumber)
	myrepo := org + slash + repo
	if err != nil {
		return "", nil, err
	}

	if len(args) > 0 {
		issueURL := args[0]
		issueRepo := GetIssueRepoFromURL(issueURL)
		if len(issueRepo) > 0 {
			parentRepo = issueRepo
		}
	}

	stdout, stderr, err := new(exec.PipedExec).
		Command("gh", "repo", "set-default", myrepo).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", nil, errors.New(stderr)
		}

		return "", nil, fmt.Errorf("failed to set default repo: %w", err)
	}
	printLn(stdout)

	branchName, err := buildDevBranchName(githubIssueURL)
	if err != nil {
		return "", nil, err
	}

	// check if a branch already exists in remote
	stdout, stderr, err = new(exec.PipedExec).
		Command(git, "ls-remote", "--heads", origin, branch).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", nil, errors.New(stderr)
		}

		return "", nil, fmt.Errorf("failed to check if branch exists in origin remote: %w", err)
	}

	if len(stdout) > 0 {
		return "", nil, fmt.Errorf("branch %s already exists in origin remote", branch)
	}

	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return "", nil, fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	stdout, stderr, err = new(exec.PipedExec).
		Command("gh", "issue", "develop", strIssueNum, "--branch-repo="+myrepo, "--repo="+parentRepo, "--name="+branchName, "--base="+mainBranch).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", nil, errors.New(stderr)
		}

		return "", nil, fmt.Errorf("failed to create development branch for issue: %w", err)
	} // delay to ensure branch is created
	logger.Verbose(stdout)

	if helper.IsTest() {
		helper.Delay()
	}

	branch = strings.TrimSpace(stdout)
	segments := strings.Split(branch, slash)
	branch = segments[len(segments)-1]

	if len(branch) == 0 {
		return "", nil, errors.New("can not create branch for issue")
	}
	// old-style notes
	issueName, err := GetIssueNameByNumber(strIssueNum, parentRepo)
	if err != nil {
		return "", nil, err
	}

	comment := IssuePRTtilePrefix + " '" + issueName + "' "
	body := ""
	if len(issueName) > 0 {
		body = IssueSign + strIssueNum + oneSpace + issueName
	}
	// Prepare new notes
	notesObj, err := notesPkg.Serialize(githubIssueURL, "", notesPkg.BranchTypeDev)
	if err != nil {
		return "", nil, err
	}

	return branch, []string{comment, body, notesObj}, nil
}

// SyncMainBranch syncs the local main branch with upstream and origin
// Flow:
// 1. Pull from UpstreamMain to MainBranch with rebase
// 2. If upstream exists
// - Pull from origin to MainBranch with rebase
// - Push to origin from MainBranch
func SyncMainBranch(wd string) error {
	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	// Pull from UpstreamMain to MainBranch with rebase
	remoteUpstreamURL := GetRemoteUpstreamURL(wd)

	if len(remoteUpstreamURL) > 0 {
		stdout, stderr, err := new(exec.PipedExec).
			Command(git, pull, "--rebase", "upstream", mainBranch, "--no-edit").
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("failed to pull from upstream/%s with rebase: %w", mainBranch, err)
		}
		logger.Verbose(stdout)
	}

	// Pull from origin to MainBranch with rebase
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, pull, "--rebase", "origin", mainBranch, "--no-edit").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to pull from origin/%s: %w with rebase", mainBranch, err)
	}
	logger.Verbose(stdout)

	// Push to origin from MainBranch
	err = helper.Retry(func() error {
		var pushErr error
		stdout, stderr, pushErr = new(exec.PipedExec).
			Command(git, push, "origin", mainBranch).
			WorkingDir(wd).
			RunToStrings()
		if pushErr != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("failed to push to origin/%s: %w", mainBranch, pushErr)
		}

		return nil
	})
	logger.Verbose(stdout)

	return err
}

// GetBranchTypeByName returns branch type based on branch name
func GetBranchTypeByName(branchName string) notesPkg.BranchType {
	switch {
	case strings.HasSuffix(branchName, "-dev"):
		return notesPkg.BranchTypeDev
	case strings.HasSuffix(branchName, "-pr"):
		return notesPkg.BranchTypePr
	default:
		return notesPkg.BranchTypeUnknown
	}
}

func buildDevBranchName(issueURL string) (string, error) {
	// Extract issue number from URL
	parts := strings.Split(issueURL, slash)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid issue URL format: %s", issueURL)
	}
	issueNumber := parts[len(parts)-1]

	// Extract owner and repo from URL
	repoURL := strings.Split(issueURL, "/issues/")[0]
	urlParts := strings.Split(repoURL, slash)
	if len(urlParts) < 5 { //nolint:revive
		return "", fmt.Errorf("invalid GitHub URL format: %s", repoURL)
	}
	owner := urlParts[3] //nolint:revive
	repo := urlParts[4]  //nolint:revive

	// Use gh CLI to get issue title
	stdout, stderr, err := new(exec.PipedExec).
		Command("gh", "issue", "view", issueNumber, "--repo", fmt.Sprintf("%s/%s", owner, repo), "--json", "title").
		RunToStrings()

	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", errors.New(stderr)
		}

		return "", fmt.Errorf("failed to get issue title: %w", err)
	}
	logger.Verbose(stdout)

	// Parse JSON response
	var issueData struct {
		Title string `json:"title"`
	}

	if err := json.Unmarshal([]byte(stdout), &issueData); err != nil {
		return "", fmt.Errorf("failed to parse issue data: %w", err)
	}

	// Create kebab-case version of the title
	kebabTitle := strings.ToLower(issueData.Title)
	// Replace spaces and special characters with dashes
	kebabTitle = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(kebabTitle, "-")
	// Remove leading and trailing dashes
	kebabTitle = strings.Trim(kebabTitle, "-")

	// Construct branch name: {issue-number}-{kebab-case-title}
	branchName := fmt.Sprintf("%s-%s", issueNumber, kebabTitle)

	// Ensure branch name doesn't exceed git's limit (usually around 250 chars)
	if len(branchName) > maximumBranchNameLength {
		branchName = branchName[:maximumBranchNameLength]
	}
	branchName = helper.CleanArgFromSpecSymbols(branchName)
	// Add suffix "-dev" for a dev branch
	branchName += "-dev"

	return branchName, nil
}

// GetBranchType returns branch type based on notes or branch name
func GetBranchType(wd string) (notesPkg.BranchType, error) {
	currentBranchName, err := GetCurrentBranchName(wd)
	if err != nil {
		return notesPkg.BranchTypeUnknown, err
	}

	notes, _, err := GetNotes(wd, currentBranchName)
	if err != nil {
		logger.Verbose(err)
	}

	if len(notes) > 0 {
		notesObj, ok := notesPkg.Deserialize(notes)
		if !ok {
			if isOldStyledBranch(notes) {
				return notesPkg.BranchTypeDev, nil
			}
		}

		if notesObj != nil {
			return notesObj.BranchType, nil
		}
	}

	return GetBranchTypeByName(currentBranchName), nil
}

// isOldStyledBranch checks if branch is old styled
func isOldStyledBranch(notes []string) bool {
	for _, s := range notes {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			if strings.Contains(s, IssuePRTtilePrefix) || strings.Contains(s, IssueSign) {
				return true
			}
		}
	}

	return false
}

func GetIssueNameByNumber(issueNum string, parentrepo string) (string, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command("gh", "issue", "view", issueNum, "--repo", parentrepo).
		Command("grep", "title:").
		Command("gawk", "{ $1=\"\"; print substr($0, 2) }").
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", errors.New(stderr)
		}

		return "", fmt.Errorf("failed to get issue name by number: %w", err)
	}
	return strings.TrimSpace(stdout), nil
}

// CreateDevBranch creates dev branch and pushes it to origin
// Parameters:
// branch - branch name
// notes - notes for branch
// checkRemoteBranchExistence - if true, checks if a branch already exists in remote
func CreateDevBranch(wd, branchName string, notes []string, checkRemoteBranchExistence bool) error {
	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "checkout", mainBranch).
		WorkingDir(wd).
		RunToStrings()

	if err != nil {
		if strings.Contains(err.Error(), err128) && strings.Contains(stderr, "matched multiple") {
			err = new(exec.PipedExec).
				Command(git, "checkout", "--track", originSlash+mainBranch).
				WorkingDir(wd).
				Run(os.Stdout, os.Stdout)
			if err != nil {
				return err
			}
		}
	}
	if err != nil {
		return err
	}

	if checkRemoteBranchExistence {
		// check if a branch already exists in remote
		stdout, stderr, err := new(exec.PipedExec).
			Command(git, "ls-remote", "--heads", "origin", branchName).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			return err
		}
		logger.Verbose(stdout)

		if len(stdout) > 0 {
			return fmt.Errorf("branch %s already exists in origin remote", branchName)
		}
	}

	// Create new branch from main
	err = new(exec.PipedExec).
		Command(git, "checkout", "-B", branchName).
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return err
	}

	// Fetch notes from origin before pushing
	stdout, stderr, err = new(exec.PipedExec).
		Command(git, fetch, origin, "--force", refsNotes).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		return fmt.Errorf("failed to fetch notes: %w, stdout: %s", err, stdout)
	}

	// Add empty commit to for keeping notes
	err = new(exec.PipedExec).
		Command(git, "commit", "--allow-empty", "-m", MsgCommitForNotes).
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return err
	}
	// Link notes to it
	if err := AddNotes(wd, notes); err != nil {
		return err
	}

	// Push notes to origin with retry
	err = helper.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, push, origin, refsNotes).
			WorkingDir(wd).
			RunToStrings()

		return err
	})
	if err != nil {
		logger.Verbose(stderr)

		return fmt.Errorf("failed to push notes to origin: %w", err)
	}
	if helper.IsTest() {
		helper.Delay()
	}

	// Push branch to origin with retry
	err = helper.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command(git, push, "-u", origin, branchName).
			WorkingDir(wd).
			RunToStrings()

		return err
	})
	if err != nil {
		logger.Verbose(stderr)

		return fmt.Errorf("failed to push branch to origin: %w, stdout: %s", err, stdout)
	}

	if helper.IsTest() {
		helper.Delay()
	}

	return nil
}

func AddNotes(wd string, notes []string) error {
	if len(notes) == 0 {
		return nil
	}
	// Add new Notes
	for _, s := range notes {
		str := strings.TrimSpace(s)
		if len(str) > 0 {
			stdout, stderr, err := new(exec.PipedExec).
				Command(git, "notes", "append", "-m", str).
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
		}
	}

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

	// get all revision of the branch which does not belong to main branch
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

	// get all notes from revisions got from a previous step
	revList := strings.Split(strings.TrimSpace(stdout), caret)
	for _, rev := range revList {
		// get notes from each revision
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
		// split notes into lines
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

// GetParentRepoName - parent repo of forked
func GetParentRepoName(wd string) (name string, err error) {
	repo, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return "", err
	}

	stdout, stderr, err := new(exec.PipedExec).
		Command("gh", "api", "repos/"+org+slash+repo, "--jq", ".parent.full_name").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", errors.New(stderr)
		}

		return "", fmt.Errorf("failed to get parent repo name: %w", err)
	}

	return strings.TrimSpace(stdout), nil
}

// IsBranchInMain Is my branch in main org?
func IsBranchInMain(wd string) (bool, error) {
	repo, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return false, err
	}
	parent, err := GetParentRepoName(wd)

	return (parent == org+slash+repo) || (strings.TrimSpace(parent) == ""), err
}

// GetMergedBranchList returns merged user's branch list
func GetMergedBranchList(wd string) (brlist []string, err error) {
	mbrlist := []string{}
	_, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return nil, fmt.Errorf("GetRepoAndOrgName failed: %w", err)
	}

	repo, err := GetParentRepoName(wd)
	if err != nil {
		return nil, fmt.Errorf("GetParentRepoName failed: %w", err)
	}

	stdout, _, err := new(exec.PipedExec).
		Command("gh", "pr", "list", "-L", "200", "--state", "merged", "--author", org, "--repo", repo).
		WorkingDir(wd).
		Command("gawk", "-F:", "{print $2}").
		Command("gawk", "/MERGED/{print $1}").
		RunToStrings()
	if err != nil {
		return []string{}, err
	}

	mbrlistraw := strings.Split(stdout, caret)
	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return nil, fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	curbr, err := GetCurrentBranchName(wd)
	if err != nil {
		return nil, err
	}

	for _, mbranchstr := range mbrlistraw {
		arrstr := strings.TrimSpace(mbranchstr)
		if (strings.TrimSpace(arrstr) != "") && !strings.Contains(arrstr, curbr) && !strings.Contains(arrstr, mainBranch) {
			mbrlist = append(mbrlist, arrstr)
		}
	}
	_, _, err = new(exec.PipedExec).
		Command(git, "remote", "prune", origin).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return nil, err
	}

	stdout, _, err = new(exec.PipedExec).
		Command(git, branch, "-r").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return nil, err
	}
	mybrlist := strings.Split(stdout, caret)

	for _, mybranch := range mybrlist {
		mybranch := strings.TrimSpace(mybranch)
		mybranch = strings.ReplaceAll(strings.TrimSpace(mybranch), originSlash, "")
		mybranch = strings.TrimSpace(mybranch)
		bfound := false
		if !strings.Contains(mybranch, mainBranch) && !strings.Contains(mybranch, "HEAD") {
			for _, mbranch := range mbrlist {
				mbranch = strings.ReplaceAll(strings.TrimSpace(mbranch), "MERGED", "")
				mbranch = strings.TrimSpace(mbranch)
				if mybranch == mbranch {
					bfound = true
					break
				}
			}
		}
		if bfound {
			// delete branch in fork
			brlist = append(brlist, mybranch)
		}
	}

	return brlist, nil
}

// DeleteBranchesRemote delete branch list
func DeleteBranchesRemote(wd string, brs []string) error {
	if len(brs) == 0 {
		return nil
	}

	for _, br := range brs {
		err := helper.Retry(func() error {
			_, _, deleteErr := new(exec.PipedExec).
				Command(git, push, origin, ":"+br).
				WorkingDir(wd).
				RunToStrings()
			return deleteErr
		})
		if err != nil {
			return fmt.Errorf("branch %s was not deleted: %w", br, err)
		}

		fmt.Printf("Branch %s deleted\n", br)
	}

	return nil
}

func PullUpstream(wd string) error {
	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	err = new(exec.PipedExec).
		Command(git, pull, "-q", "upstream", mainBranch, "--no-edit").
		Run(os.Stdout, os.Stdout)

	if err != nil {
		parentRepoName, err := GetParentRepoName(wd)
		if err != nil {
			return fmt.Errorf("GetParentRepoName failed: %w", err)
		}

		return MakeUpstreamForBranch(wd, parentRepoName)
	}

	return nil
}

// GetGoneBranchesLocal returns gone local branches
func GetGoneBranchesLocal(wd string) (*[]string, error) {
	// https://dev.heeus.io/launchpad/#!14544
	// 1. Step
	_, _, err := new(exec.PipedExec).
		Command(git, fetch, "-p", "--dry-run").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return nil, err
	}
	_, _, err = new(exec.PipedExec).
		Command(git, fetch, "-p").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return nil, err
	}
	// 2. Step
	stdout, _, err := new(exec.PipedExec).
		Command(git, branch, "-vv").
		WorkingDir(wd).
		Command("grep", "\\[[^]]*: gone[^]]*\\]").
		Command("gawk", "{print $1}").
		RunToStrings()
	if err != nil {
		return nil, err
	}
	// alternate: grep '\[.*: gone'
	mbrlocallist := strings.Split(stdout, caret)

	stsr := []string{}
	stdout, _, err = new(exec.PipedExec).
		Command(git, branch, "-r").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return nil, err
	}
	myremotelist := strings.Split(stdout, caret)
	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return nil, fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	curbr, err := GetCurrentBranchName(wd)
	if err != nil {
		return nil, err
	}

	for _, mylocalbranch := range mbrlocallist {
		mybranch := strings.TrimSpace(mylocalbranch)
		bfound := false
		if strings.Contains(mybranch, curbr) {
			bfound = true
		} else if !strings.Contains(mybranch, mainBranch) && !strings.Contains(mybranch, "HEAD") {
			for _, mbranch := range myremotelist {
				mbranch = strings.TrimSpace(mbranch)
				if mybranch == mbranch {
					bfound = true
					break
				}
			}
		}
		if !bfound {
			// delete branch in fork
			stsr = append(stsr, mybranch)
		}
	}
	return &stsr, nil
}

// DeleteBranchesLocal s.e.
func DeleteBranchesLocal(wd string, strs *[]string) error {
	for _, str := range *strs {
		if strings.TrimSpace(str) != "" {
			_, _, err := new(exec.PipedExec).
				Command(git, branch, "-D", str).
				WorkingDir(wd).
				RunToStrings()
			fmt.Printf("Branch %s deleted\n", str)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func GetNoteAndURL(notes []string) (note string, url string) {
	for _, s := range notes {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			if strings.Contains(s, httppref) {
				url = s
				if len(note) > 0 {
					break
				}
			} else {
				if note == "" {
					note = s
				} else {
					note = note + oneSpace + s
				}
				if strings.Contains(strings.ToLower(s), strings.ToLower(IssuePRTtilePrefix)) {
					break
				}
			}
		}
	}
	return note, url
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

func createPR(wd, parentRepoName, prBranchName string, notes []string, asDraft bool) (stdout string, stderr string, err error) {
	if len(notes) == 0 {
		return "", "", errors.New(ErrMsgPRNotesImpossible)
	}

	//ParseGitRemoteURL()
	// get json notes object from dev branch
	notesObj, ok := notesPkg.Deserialize(notes)
	if !ok {
		return "", "", errors.New("error deserializing notes")
	}

	var prTitle string
	var isCustomBranch bool
	switch {
	case len(notesObj.GithubIssueURL) > 0:
		prTitle, err = GetIssueDescription(notesObj.GithubIssueURL)
	case len(notesObj.JiraTicketURL) > 0:
		prTitle, err = jira.GetJiraIssueName(notesObj.JiraTicketURL, "")
	default:
		isCustomBranch = true
		fmt.Print("Enter pull request title: ")
		reader := bufio.NewReader(os.Stdin)

		// Read until newline (includes spaces)
		prTitle, err = reader.ReadString(caretByte)
		if err != nil {
			return "", "", err
		}

		prTitle = strings.TrimSpace(prTitle)
		if len(prTitle) < minPRTitleLength {
			return "", "", errors.New("too short pull request title")
		}
	}
	if err != nil {
		return "", "", fmt.Errorf("error retrieving pull request title: %w", err)
	}

	var strNotes string
	var url string
	strNotes, url = GetNoteAndURL(notes)
	b := GetBodyFromNotes(notes)
	if len(b) == 0 && !isCustomBranch {
		b = strNotes
	}
	if len(url) > 0 {
		b = b + caret + url
	}
	strBody := fmt.Sprintln(b)

	repoName, forkAccount, err := GetRepoAndOrgName(wd)
	if err != nil {
		return "", "", err
	}

	repo := parentRepoName
	if len(repo) == 0 {
		repo = forkAccount + slash + repoName
	}

	args := []string{
		"pr",
		"create",
		fmt.Sprintf(`--head=%s`, forkAccount+":"+prBranchName),
		fmt.Sprintf(`--repo=%s`, repo),
		fmt.Sprintf(`--body=%s`, strings.TrimSpace(strBody)),
		fmt.Sprintf(`--title=%s`, strings.TrimSpace(prTitle)),
	}
	if asDraft {
		args = append(args, "--draft")
	}
	err = helper.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command("gh", args...).
			RunToStrings()

		return err
	})
	if err != nil {
		return stdout, stderr, err
	}

	prInfo, stdout, stderr, err := DoesPrExist(wd, parentRepoName, prBranchName, PRStateOpen)
	if err != nil {
		return stdout, stderr, err
	}
	if prInfo == nil {
		return stdout, stderr, errors.New("PR not created")
	}
	// print PR URL
	if len(prInfo.URL) > 0 {
		fmt.Println()
		fmt.Println(prInfo.URL)
	}

	return stdout, stderr, err
}

func retrieveRepoNameFromUPL(prurl string) string {
	var strs = strings.Split(prurl, slash)
	if len(strs) < minRepoNameLength {
		return ""
	}
	res := ""
	lenstr := len(strs)
	for i := lenstr - minRepoNameLength; i < lenstr-2; i++ {
		if res == "" {
			res = strs[i]
		} else {
			res = res + slash + strs[i]
		}
	}
	return res
}

// getRemotes shows list of names of all remotes
func getRemotes(wd string) []string {
	stdout, _, _ := new(exec.PipedExec).
		Command(git, "remote").
		WorkingDir(wd).
		RunToStrings()
	strs := strings.Split(stdout, caret)
	for i, str := range strs {
		if len(strings.TrimSpace(str)) == 0 {
			strs = append(strs[:i], strs[i+1:]...)
		}
	}
	return strs
}

// GetFilesForCommit shows list of file names, ready for commit
func GetFilesForCommit(wd string) []string {
	stdout, _, _ := new(exec.PipedExec).
		Command(git, "status", "-s").
		WorkingDir(wd).
		RunToStrings()
	ss := strings.Split(stdout, caret)
	var strs []string
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			strs = append(strs, s)
		}
	}
	return strs
}

// hasUpstreamBranch checks if the current branch has an upstream tracking branch configured
func hasUpstreamBranch(wd string, branchName string) (bool, error) {
	stdout, _, err := new(exec.PipedExec).
		Command(git, "config", "--get", fmt.Sprintf("branch.%s.remote", branchName)).
		WorkingDir(wd).
		RunToStrings()

	if err != nil {
		// If the config doesn't exist, git config returns exit code 1
		// This is expected when no upstream is configured
		return false, nil
	}

	return strings.TrimSpace(stdout) != "", nil
}

// hasRemoteTrackingBranch checks if a remote tracking branch exists for the given branch
func hasRemoteTrackingBranch(wd string, branchName string) (bool, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "branch", "-r").
		WorkingDir(wd).
		Command("grep", branchName).
		RunToStrings()
	if len(stdout) == 0 {
		return false, nil
	}

	if err != nil {
		logger.Verbose(stderr)

		return false, err
	}

	strBranches := strings.TrimSpace(stdout)
	if strBranches == "" {
		return false, nil
	}

	rBranches := strings.Split(strBranches, "\n")
	for _, rBranch := range rBranches {
		if strings.TrimSpace(rBranch) == "origin/"+branchName {
			return true, nil
		}
	}

	return false, nil
}

func getGlobalHookFolder() string {
	stdout, _, err := new(exec.PipedExec).
		Command(git, "config", "--global", "core.hooksPath").
		RunToStrings()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdout)
}

func getLocalHookFolder(wd string) (string, error) {
	dir, err := GetRootFolder(wd)
	if err != nil {
		return "", err
	}
	filename := "/.git/hooks/pre-commit"
	filepath := dir + filename

	return strings.TrimSpace(filepath), nil
}

// GlobalPreCommitHookExist - s.e.
func GlobalPreCommitHookExist() (bool, error) {
	filepath := getGlobalHookFolder()
	if len(filepath) == 0 {
		return false, nil // global hook folder not defined
	}
	err := os.MkdirAll(filepath, os.ModePerm)
	if err != nil {
		return false, err
	}

	filepath += "/pre-commit"
	// Check if the file already exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return false, nil // File pre-commit does not exist
	}

	return largeFileHookExist(filepath), nil
}

// LocalPreCommitHookExist - s.e.
func LocalPreCommitHookExist(wd string) (bool, error) {
	filepath, err := getLocalHookFolder(wd)
	if err != nil {
		return false, err
	}
	// Check if the file already exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return false, nil
	}

	return largeFileHookExist(filepath), nil
}

func largeFileHookExist(filepath string) bool {
	substring := LargeFileHookFilename
	_, _, err := new(exec.PipedExec).Command("grep", "-l", substring, filepath).RunToStrings()

	return err == nil
}

// SetGlobalPreCommitHook - s.e.
func SetGlobalPreCommitHook(wd string) error {
	var err error
	path := getGlobalHookFolder()

	if len(path) == 0 {
		rootUser, err := user.Current()
		if err != nil {
			return err
		}

		path = rootUser.HomeDir
		path += "/.git/hooks"
		if err = os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
	}

	// Set global hooks folder
	err = new(exec.PipedExec).
		Command(git, "config", "--global", "core.hookspath", path).
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return err
	}

	filepath := path + "/pre-commit"
	f, err := createOrOpenFile(filepath)
	if err != nil {
		return err
	}

	_ = f.Close()
	if !largeFileHookExist(filepath) {
		return fillPreCommitFile(wd, filepath)
	}

	return nil
}

func GetRootFolder(wd string) (string, error) {
	stdout, _, err := new(exec.PipedExec).
		Command(git, "rev-parse", "--show-toplevel").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout), nil
}

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

func UpstreamNotExist(wd string) bool {
	return len(getRemotes(wd)) < 2
}

func HasRemote(wd, remoteName string) (bool, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "remote").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return false, errors.New(stderr)
		}

		return false, fmt.Errorf("failed to list git remotes: %w: %s", err, stderr)
	}

	remotes := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, remote := range remotes {
		if strings.TrimSpace(remote) == remoteName {
			return true, nil
		}
	}

	return false, nil
}

func GawkInstalled() bool {
	_, _, err := new(exec.PipedExec).
		Command("gawk", "--version").
		RunToStrings()
	return err == nil
}

func extractIntegerPrefix(input string) (string, error) {
	// Define the regular expression pattern
	pattern := `^\d+`
	re := regexp.MustCompile(pattern)

	// Find the match
	match := re.FindString(input)
	if match == "" {
		return "", fmt.Errorf("no integer found at the beginning of the string")
	}

	// Convert the matched string to an integer
	integerValue, err := strconv.Atoi(match)
	if err != nil {
		return "", fmt.Errorf("error converting string to integer: %w", err)
	}

	return strconv.Itoa(integerValue), nil
}

func issuenumExists(parentrepo string, issuenum string) bool {
	stdouts, _, err := new(exec.PipedExec).
		Command("gh", "issue", "develop", issuenum, "--list", "-R", parentrepo).
		Command("gawk", "{print $2}").
		RunToStrings()
	if (err == nil) && (len(stdouts) > minIssueNoteLength) {
		names := strings.Split(stdouts, caret)
		for _, name := range names {
			if strings.Contains(name, slash+issuenum+"-") {
				return true
			}
		}
	}
	return false
}

func GetIssueNumFromBranchName(parentrepo string, curbranch string) (issuenum string, ok bool) {

	tempissuenum, err := extractIntegerPrefix(curbranch)
	if tempissuenum == "" {
		return "", false
	}
	if err == nil {
		if issuenumExists(parentrepo, tempissuenum) {
			return tempissuenum, true
		}
	}

	stdouts, stderr, err := new(exec.PipedExec).
		Command("gh", "issue", "list", "-R", parentrepo).
		Command("gawk", "{print $1}").
		RunToStrings()
	if err != nil {
		logger.Verbose("GetIssueNumFromBranchName:", stderr)
		return "", false
	}
	issuenums := strings.Split(stdouts, caret)
	fmt.Println("Searching linked issue ")

	for _, issuenum := range issuenums {
		if len(issuenum) > 0 {
			fmt.Println("  Issue number: ", issuenum, "...")
			if issuenumExists(parentrepo, issuenum) {
				return issuenum, true
			}
		}
	}

	return "", false
}

func LinkIssueToMileStone(issueNum string, parentrepo string) error {
	if issueNum == "" {
		return nil
	}
	if parentrepo == "" {
		return nil
	}
	stdout, stderr, err := new(exec.PipedExec).
		Command("gh", "api", "repos/"+parentrepo+"/milestones", "--jq", ".[] | .title").
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("link issue to mileStone error: %w", err)
	}

	milestones := strings.Split(stdout, caret)
	// Sample date string in the "yyyy.mm.dd" format.
	dateString := "2006.01.02"
	// Get the current date and time.
	currentTime := time.Now()
	for _, milestone := range milestones {
		// Parse the input string into a time.Time value.
		t, err := time.Parse(dateString, milestone)
		if err == nil {
			if currentTime.Before(t) {
				// Next milestone is found
				stdout, stderr, err = new(exec.PipedExec).
					Command("gh", "issue", "edit", issueNum, "--milestone", milestone, "--repo", parentrepo).
					RunToStrings()
				if err != nil {
					logger.Verbose(stderr)

					if len(stderr) > 0 {
						return errors.New(stderr)
					}

					return fmt.Errorf("failed to link issue to milestone: %w", err)
				}
				logger.Verbose(stdout)
				fmt.Println("Issue #" + issueNum + " added to milestone '" + milestone + "'")

				return nil
			}
		}
	}

	return nil
}

func GetCurrentBranchName(wd string) (string, error) {
	branchName, stderr, err := new(exec.PipedExec).
		Command(git, branch, "--show-current").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", errors.New(stderr)
		}

		return "", fmt.Errorf("failed to get current branch name: %w", err)
	}

	return strings.TrimSpace(branchName), nil
}

// createRemote creates a remote in the cloned repository
func CreateRemote(wd, remote, account, token, repoName string, isUpstream bool) error {
	repo, err := goGitPkg.PlainOpen(wd)
	if err != nil {
		return fmt.Errorf("failed to open cloned repository: %w", err)
	}

	if err = repo.DeleteRemote(remote); err != nil {
		if !errors.Is(err, goGitPkg.ErrRemoteNotFound) {
			return fmt.Errorf("failed to delete %s remote: %w", remote, err)
		}
	}

	remoteURL := BuildRemoteURL(account, token, repoName, isUpstream)
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: remote,
		URLs: []string{remoteURL},
	})
	if err != nil {
		return fmt.Errorf("failed to create %s remote: %w", remote, err)
	}

	return nil
}

// buildRemoteURL constructs the remote URL for cloning
func BuildRemoteURL(account, token, repoName string, isUpstream bool) string {
	return "https://" + account + ":" + token + "@github.com/" + account + slash + repoName + ".git"
}

// IamInMainBranch checks if current branch is main branch
// Returns:
// - the name of current branch
// - true if current branch is main branch
// - error if any
func IamInMainBranch(wd string) (string, bool, error) {
	currentBranchName, err := GetCurrentBranchName(wd)
	if err != nil {
		return "", false, err
	}
	logger.Verbose("Current branch: " + currentBranchName)

	mainBranch, err := GetMainBranch(wd)
	logger.Verbose("Main branch: " + mainBranch)
	if err != nil {
		return "", false, fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	return currentBranchName, strings.EqualFold(currentBranchName, mainBranch), err
}

// GetRemoteUrlByName retrieves the URL of a specified remote by its name
func GetRemoteUrlByName(wd string, remoteName string) (string, error) {
	repo, err := goGitPkg.PlainOpen(wd)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	remotes, err := repo.Remotes()
	if err != nil {
		return "", fmt.Errorf("failed to get remotes: %w", err)
	}

	for _, remote := range remotes {
		if remote.Config().Name == remoteName {
			if len(remote.Config().URLs) > 0 {
				return remote.Config().URLs[0], nil
			}
			return "", fmt.Errorf("remote %s has no URLs configured", remoteName)
		}
	}

	return "", fmt.Errorf("remote %s not found", remoteName)
}

func printLn(stdout string) {
	if len(stdout) > 0 {
		fmt.Println(stdout)
	}
}
