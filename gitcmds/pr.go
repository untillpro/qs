package gitcmds

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/internal/helper"
	"github.com/untillpro/qs/internal/jira"
	notesPkg "github.com/untillpro/qs/internal/notes"
)

type PRInfo struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func Pr(wd string, needDraft bool) error {
	// find out the type of the branch
	branchType, err := GetBranchType(wd)
	if err != nil {
		return err
	}

	logger.Verbose(fmt.Sprintf("branch type is %s", branchType.String()))
	if branchType == notesPkg.BranchTypeUnknown {
		return errors.New("you must be on dev or pr branch")
	}

	currentBranchName, err := GetCurrentBranchName(wd)
	if err != nil {
		return err
	}

	parentRepoName, err := GetParentRepoName(wd)
	if err != nil {
		return err
	}
	if len(parentRepoName) == 0 {
		return errors.New("you are in trunk. PR is only allowed from forked branch")
	}

	// If we are on dev branch than we need to create pr branch
	if branchType == notesPkg.BranchTypeDev {
		// Fetch notes from origin before checking if they exist
		_, _, err := new(exec.PipedExec).
			Command(git, fetch, origin, "--force", "refs/notes/*:refs/notes/*").
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose("Failed to fetch notes: %v", err)
			// Continue anyway, as notes might exist locally
		}

		var response string
		if UpstreamNotExist(wd) {
			fmt.Print("Upstream not found.\nRepository " + parentRepoName + " will be added as upstream. Agree[y/n]?")
			_, _ = fmt.Scanln(&response)
			if response != pushYes {
				fmt.Print(msgOkSeeYou)
				return nil
			}
			response = ""
			if err := MakeUpstreamForBranch(wd, parentRepoName); err != nil {
				return fmt.Errorf("failed to set upstream: %w", err)
			}
		}

		// Check if there are any modified files in the current branch
		if _, ok, err := ChangedFilesExist(wd); ok || err != nil {
			if err != nil {
				return err
			}

			return errors.New(errMsgModFiles)
		}

		// Create a new branch for the PR
		prBranchName, err := createPRBranch(wd, currentBranchName)
		if err != nil {
			return fmt.Errorf("failed to create PR branch: %w", err)
		}

		// Remove dev branch after creating PR-branch
		if err := RemoveBranch(wd, currentBranchName); err != nil {
			logger.Verbose(fmt.Errorf("failed to remove branch: %w", err))
		}

		// Current branch now is pr branch
		currentBranchName = prBranchName
	}

	// push notes and commits to origin
	if err := pushPRBranch(wd, currentBranchName); err != nil {
		return err
	}

	// Check whether PR already exists
	prExists, _, _, _, err := DoesPrExist(wd, parentRepoName, currentBranchName, PRStateOpen)
	if err != nil {
		return err
	}
	if prExists {
		fmt.Fprintln(os.Stdout, "pull request already exists for this branch")

		return nil
	}

	// Extract notes before any operations
	notes, revCount, err := GetNotes(wd, currentBranchName)
	if err != nil {
		return err
	}

	if revCount == 0 {
		return errors.New("error: No commits found in pr branch")
	}

	// Create PR
	stdout, stderr, err := createPR(wd, parentRepoName, currentBranchName, notes, needDraft)
	if err != nil {
		logger.Verbose(stdout)
		logger.Verbose(stderr)

		return fmt.Errorf("failed to create PR: %w", err)
	}

	return nil
}

// pushPRBranch pushes the PR branch to origin.
func pushPRBranch(wd, prBranchName string) error {
	// Push notes to origin
	err := helper.Retry(func() error {
		return new(exec.PipedExec).
			Command("git", "push", "origin", "refs/notes/*:refs/notes/*").
			WorkingDir(wd).
			Run(os.Stdout, os.Stdout)
	})
	if err != nil {
		return fmt.Errorf("failed to push notes to origin: %w", err)
	}

	if helper.IsTest() {
		helper.Delay()
	}

	// Push PR branch to origin
	err = helper.Retry(func() error {
		_, _, pushErr := new(exec.PipedExec).
			Command("git", "push", "-u", "origin", prBranchName).
			WorkingDir(wd).
			RunToStrings()
		return pushErr
	}) // Retry up to 3 times for pushing PR branch
	if err != nil {
		return err
	}

	if helper.IsTest() {
		helper.Delay()
	}
	// Push PR branch to origin
	err = helper.Retry(func() error {
		_, _, pushErr := new(exec.PipedExec).
			Command("git", "push", "-u", "origin", prBranchName).
			WorkingDir(wd).
			RunToStrings()
		return pushErr
	})
	if err != nil {
		return err
	}

	return nil
}

// DoesPrExist checks if a pull request exists for the current branch.
// Returns:
// - true if PR exists
// - PRInfo object if PR exists, nil otherwise
// - stdout from the command execution
// - stderr from the command execution
// - error if any
func DoesPrExist(wd, parentRepo, currentBranchName string, prState PRState) (bool, *PRInfo, string, string, error) {
	var (
		prExists bool
		prInfo   PRInfo
		stdout   string
		stderr   string
		err      error
	)

	err = helper.Retry(func() error {
		stdout, stderr, err = new(exec.PipedExec).
			Command(
				"gh",
				"pr",
				"list",
				"--repo",
				parentRepo,
				"--head",
				currentBranchName,
				"--limit",
				"1",
				"--state",
				string(prState),
				"--json",
				"url,title",
			).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			return err
		}

		if strings.Contains(stdout, "no pull requests match your search") {
			prExists = false

			return nil
		}

		// Otherwise, assume PR exists
		if stdout == "" {
			// safety check: gh may sometimes return nothing at all
			prExists = false

			return nil
		}
		var infos []PRInfo
		if err := json.Unmarshal([]byte(stdout), &infos); err != nil {
			return fmt.Errorf("failed to parse gh pr list output: %w", err)
		}
		if len(infos) > 0 {
			prInfo.URL = strings.TrimSpace(infos[0].URL)
			prInfo.Title = strings.TrimSpace(infos[0].Title)
			prExists = true
		}

		return nil
	})
	if err != nil {
		return false, nil, stdout, stderr, err
	}

	return prExists, &prInfo, stdout, stderr, nil
}

func RemoveBranch(wd, branchName string) error {
	// Delete branch locally
	_, stderr, err := new(exec.PipedExec).
		Command("git", "branch", "-D", branchName).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		return err
	}

	// Delete branch from origin
	err = helper.Retry(func() error {
		_, stderr, err = new(exec.PipedExec).
			Command("git", "push", "origin", "--delete", branchName).
			WorkingDir(wd).
			RunToStrings()

		return err
	})
	if err != nil {
		logger.Verbose(stderr)

		return err
	}

	return nil
}

// createPRBranch creates a new branch for the pull request and checks out on it.
// Returns:
// - name of the PR branch
// - error if any operation fails
func createPRBranch(wd, devBranchName string) (string, error) {
	// Save current branch name (dev branch)
	prBranchName := strings.TrimSuffix(devBranchName, "-dev") + "-pr"

	upstreamRemote := "upstream"
	upstreamExists, err := HasRemote(wd, upstreamRemote)
	if err != nil {
		return "", err
	}

	mainBranchName, err := GetMainBranch(wd)
	if err != nil {
		return "", fmt.Errorf("failed to get main branch: %w", err)
	}

	if !upstreamExists {
		upstreamRemote = "origin"
	}
	upstreamMain := upstreamRemote + "/" + mainBranchName

	// Step 1: Fetch latest upstream
	err = helper.Retry(func() error {
		_, _, fetchErr := new(exec.PipedExec).
			Command("git", "fetch", upstreamRemote).
			WorkingDir(wd).
			RunToStrings()
		return fetchErr
	}) // Retry up to 3 times for fetching upstream
	if err != nil {
		return "", err
	}

	// Step 1.1: Fetch notes from origin
	var stdout, stderr string
	err = helper.Retry(func() error {
		var fetchErr error
		stdout, stderr, fetchErr = new(exec.PipedExec).
			Command("git", "fetch", "origin", "--force", "refs/notes/*:refs/notes/*").
			WorkingDir(wd).
			RunToStrings()
		return fetchErr
	}) // Retry up to 3 times for fetching notes
	if err != nil {
		logger.Verbose(stdout)
		logger.Verbose(stderr)

		return "", fmt.Errorf("failed to fetch notes: %w", err)
	}

	// Step 2: Checkout dev branch

	// extract notes from dev branch before any operations
	notes, revCount, err := GetNotes(wd, devBranchName)
	if err != nil {
		return "", err
	}

	// if we have only 1 revision in dev branch then it is just a commit for keeping notes
	if revCount < 2 {
		return "", errors.New("error: No commits found in dev branch")
	}

	_, _, err = new(exec.PipedExec).
		Command("git", "checkout", devBranchName).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return "", err
	}

	// Step 3: Merge from origin/main + upstream/main
	_, _, err = new(exec.PipedExec).
		Command("git", "merge", "origin/"+mainBranchName).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return "", err
	}

	// Step 3.1
	if upstreamExists {
		_, _, err = new(exec.PipedExec).
			Command("git", "merge", "upstream/"+mainBranchName).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			return "", err
		}
	}

	// Step 4: Create new PR branch from upstream/main
	_, _, err = new(exec.PipedExec).
		Command("git", "checkout", "-b", prBranchName, upstreamMain).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return "", err
	}

	// Step 5: Squash merge dev into PR branch
	_, _, err = new(exec.PipedExec).
		Command("git", "merge", "--squash", devBranchName).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return "", err
	}

	// Step 6: Get issue description from notes for commit message

	// get json notes object from dev branch
	notesObj, ok := notesPkg.Deserialize(notes)
	if !ok {
		return "", errors.New("error deserializing notes")
	}
	// update branch type in notes object
	notesObj.BranchType = notesPkg.BranchTypePr

	var description string
	switch {
	case len(notesObj.GithubIssueURL) > 0:
		description, err = GetIssueDescription(notesObj.GithubIssueURL)
	case len(notesObj.JiraTicketURL) > 0:
		description, err = jira.GetJiraIssueName(notesObj.JiraTicketURL, "")
	default:
		description = DefaultCommitMessage
	}

	if err != nil {
		return "", fmt.Errorf("error retrieving issue description: %w", err)
	}

	// Step 7: Commit the squashed changes
	stdout, stderr, err = new(exec.PipedExec).
		Command("git", "commit", "-m", description).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stdout)
		logger.Verbose(stderr)

		return "", err
	}

	// Add empty commit to create commit object and link notes to it
	if err := AddNotes(wd, updateNotesObjInNoteLines(notes, *notesObj)); err != nil {
		return "", err
	}

	return prBranchName, nil
}

// GetIssueDescription retrieves the title and body of a GitHub issue from its URL.
func GetIssueDescription(issueURL string) (string, error) {
	// Extract issue number from URL
	parts := strings.Split(issueURL, "/")
	if len(parts) < 1 {
		return "", fmt.Errorf("invalid issue URL format: %s", issueURL)
	}
	issueNumber := parts[len(parts)-1]

	// Extract owner and repo from URL
	repoURL, err := convertIssuesURLToRepoURL(issueURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse repo from issue URL: %w", err)
	}

	urlParts := strings.Split(repoURL, "/")
	if len(urlParts) < 5 {//nolint:revive
		return "", fmt.Errorf("invalid GitHub URL format: %s", repoURL)
	}
	owner := urlParts[3]//nolint:revive
	repo := urlParts[4]//nolint:revive

	// Use gh CLI to get issue details in JSON format with retry logic
	var issueData struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	err = helper.Retry(func() error {
		stdout, stderr, err := new(exec.PipedExec).
			Command("gh", "issue", "view", issueNumber, "--repo", fmt.Sprintf("%s/%s", owner, repo), "--json", "title,body").
			RunToStrings()

		if err != nil {
			return fmt.Errorf("failed to get issue: %w, stderr: %s", err, stderr)
		}

		// Parse JSON response
		if err := json.Unmarshal([]byte(stdout), &issueData); err != nil {
			return fmt.Errorf("failed to parse issue data: %w", err)
		}

		return nil
	}) // Retry up to 3 times for issue retrieval

	if err != nil {
		return "", err
	}

	return issueData.Title, nil
}

func convertIssuesURLToRepoURL(issueURL string) (string, error) {
	// Remove trailing issue number
	issuesParts := strings.Split(issueURL, "/issues/")
	if len(issuesParts) < 2 {
		return "", fmt.Errorf("invalid GitHub issue URL format: %s", issueURL)
	}

	repoURL := issuesParts[0]
	return repoURL, nil
}

// updateNotesObjInNoteLines updates notes by removing old notes and adding new ones
func updateNotesObjInNoteLines(notes []string, notesObj notesPkg.Notes) []string {
	newNotes := make([]string, 0, len(notes))
	for _, s := range notes {
		if strings.Contains(s, "{") {
			continue
		}
		newNotes = append(newNotes, s)
	}

	return append(newNotes, notesObj.String())
}
