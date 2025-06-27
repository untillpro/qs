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
	notesPkg "github.com/untillpro/qs/internal/notes"
)

func Pr(wd string, needDraft bool) error {
	// find out type of the branch
	branchType := GetBranchType(wd)
	if branchType != notesPkg.BranchTypeDev {
		return errors.New("You must be on dev branch")
	}

	// PR is not created yet
	prExists, err := doesPrExist(wd)
	if err != nil {
		return err
	}
	if prExists {
		return errors.New("Pull request already exists for this branch")
	}

	parentrepo, err := GetParentRepoName(wd)
	if err != nil {
		return err
	}
	if len(parentrepo) == 0 {
		return errors.New("You are in trunk. PR is only allowed from forked branch.")
	}
	curBranch := GetCurrentBranchName(wd)
	isMainBranch := (curBranch == "main") || (curBranch == "master")
	if isMainBranch {
		return fmt.Errorf("Unable to create a pull request on branch '%s'. Use 'qs dev <branch_name>.", curBranch)
	}

	var response string
	if UpstreamNotExist(wd) {
		fmt.Print("Upstream not found.\nRepository " + parentrepo + " will be added as upstream. Agree[y/n]?")
		_, _ = fmt.Scanln(&response)
		if response != pushYes {
			fmt.Print(pushFail)
			return nil
		}
		response = ""
		if err := MakeUpstreamForBranch(wd, parentrepo); err != nil {
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
	prTitle, err := createPRBranch(wd)
	if err != nil {
		return fmt.Errorf("failed to create PR branch: %w", err)
	}
	// Extract notes before any operations
	notes, ok := GetNotes(wd)
	if !ok {
		return errors.New("Error: No notes found in dev branch")
	}

	stdout, stderr, err := MakePR(wd, prTitle, notes, needDraft)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stdout, stdout)
		_, _ = fmt.Fprintln(os.Stderr, stderr)

		return fmt.Errorf("failed to create PR: %w", err)
	}

	return nil
}

// doesPrExist checks if a pull request exists for the current branch.
func doesPrExist(wd string) (bool, error) {
	branchName := GetCurrentBranchName(wd)
	stdout, _, err := new(exec.PipedExec).
		Command("gh", "pr", "list", "--head", branchName).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return false, err
	}

	if strings.Contains(stdout, "no pull requests match your search") {
		return false, nil
	}

	// Otherwise, assume PR exists
	if stdout == "" {
		// safety check: gh may sometimes return nothing at all
		return false, nil
	}

	return true, nil
}

// createPRBranch creates a new branch for the pull request and checks out on it.
// Returns:
// - title of the pull request
// - error if any operation fails
func createPRBranch(wd string) (string, error) {
	// Save current branch name (dev branch)
	devBranchName := GetCurrentBranchName(wd)
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
	_, _, err = new(exec.PipedExec).
		Command("git", "fetch", upstreamRemote).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return "", err
	}

	// Step 1.1: Fetch notes from origin
	stdout, stderr, err := new(exec.PipedExec).
		Command("git", "fetch", "origin", "refs/notes/*:refs/notes/*").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Error(stdout)
		logger.Error(stderr)

		return "", fmt.Errorf("failed to fetch notes: %w", err)
	}

	// Step 2: Checkout dev branch

	// extract notes from dev branch before any operations
	notes, ok := GetNotes(wd)
	if !ok {
		return "", errors.New("Error: No notes found in dev branch")
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

	issueDescription, err := getIssueDescription(notesObj.GithubIssueURL)
	if err != nil {
		return "", fmt.Errorf("Error retrieving issue description: %w", err)
	}

	// Step 7: Commit the squashed changes
	stdout, stderr, err = new(exec.PipedExec).
		Command("git", "commit", "-m", issueDescription).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Error(stdout)
		logger.Error(stderr)

		return "", err
	}

	// Add empty commit to create commit object and link notes to it
	if err := AddNotes(wd, updateNotesObjInNoteLines(notes, *notesObj)); err != nil {
		return "", err
	}

	// Step 8: Push notes to origin
	err = new(exec.PipedExec).
		Command("git", "push", "origin", "refs/notes/*:refs/notes/*").
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return "", err
	}

	if helper.IsTest() {
		helper.Delay()
	}

	// Step 9: Push PR branch to origin
	_, _, err = new(exec.PipedExec).
		Command("git", "push", "-u", "origin", prBranchName).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return "", err
	}

	if helper.IsTest() {
		helper.Delay()
	}

	// Step 10: Delete dev branch locally
	_, _, err = new(exec.PipedExec).
		Command("git", "branch", "-D", devBranchName).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return "", err
	}

	// Step 11: Delete dev branch from origin
	_, _, err = new(exec.PipedExec).
		Command("git", "push", "origin", "--delete", devBranchName).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return "", err
	}

	// Step 11: Delete dev branch from upstream
	_, stderr, err = new(exec.PipedExec).
		Command("git", "push", "upstream", "--delete", devBranchName).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		if strings.Contains(stderr, "ref does not exist") {
			return issueDescription, nil
		}

		return "", err
	}

	return issueDescription, nil
}

// getIssueDescription retrieves the title and body of a GitHub issue from its URL.
func getIssueDescription(issueURL string) (string, error) {
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
	if len(urlParts) < 5 {
		return "", fmt.Errorf("invalid GitHub URL format: %s", repoURL)
	}
	owner := urlParts[3]
	repo := urlParts[4]

	// Use gh CLI to get issue details in JSON format
	stdout, stderr, err := new(exec.PipedExec).
		Command("gh", "issue", "view", issueNumber, "--repo", fmt.Sprintf("%s/%s", owner, repo), "--json", "title,body").
		RunToStrings()

	if err != nil {
		return "", fmt.Errorf("failed to get issue: %w, stderr: %s", err, stderr)
	}

	// Parse JSON response
	var issueData struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	if err := json.Unmarshal([]byte(stdout), &issueData); err != nil {
		return "", fmt.Errorf("failed to parse issue data: %w", err)
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
