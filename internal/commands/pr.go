package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/qs/git"
	"github.com/untillpro/qs/internal/commands/helper"
	notesPkg "github.com/untillpro/qs/internal/notes"
	"github.com/untillpro/qs/internal/types"
)

func Pr(cmd *cobra.Command) error {
	globalConfig()
	if _, err := git.CheckIfGitRepo(); err != nil {
		return err
	}

	if !helper.CheckQsVer() {
		return fmt.Errorf("qs version check failed")
	}
	if !helper.CheckGH() {
		return fmt.Errorf("GitHub CLI check failed")
	}

	// find out type of the branch
	branchType := git.GetBranchType()
	if branchType != types.BranchTypeDev {
		return errors.New("You must be on dev branch")
	}

	// PR is not created yet
	prExists := doesPrExist()
	if prExists {
		return errors.New("Pull request already exists for this branch")
	}

	parentrepo := git.GetParentRepoName()
	if len(parentrepo) == 0 {
		return errors.New("You are in trunk. PR is only allowed from forked branch.")
	}
	curBranch := git.GetCurrentBranchName()
	isMainBranch := (curBranch == "main") || (curBranch == "master")
	if isMainBranch {
		return fmt.Errorf("Unable to create a pull request on branch '%s'. Use 'qs dev <branch_name>.", curBranch)
	}

	var response string
	if git.UpstreamNotExist() {
		fmt.Print("Upstream not found.\nRepository " + parentrepo + " will be added as upstream. Agree[y/n]?")
		_, _ = fmt.Scanln(&response)
		if response != pushYes {
			fmt.Print(pushFail)
			return nil
		}
		response = ""
		if err := git.MakeUpstreamForBranch(parentrepo); err != nil {
			return fmt.Errorf("failed to set upstream: %w", err)
		}
	}

	if git.PRAhead() {
		fmt.Print("This branch is out-of-date. Merge automatically[y/n]?")
		_, _ = fmt.Scanln(&response)
		if response != pushYes {
			fmt.Print(pushFail)
			return nil
		}
		response = ""
		git.MergeFromUpstreamRebase()
	}

	// Check if there are any modified files in the current branch
	if _, ok, err := git.ChangedFilesExist(); ok || err != nil {
		if err != nil {
			return err
		}

		return errors.New(errMsgModFiles)
	}

	// Create a new branch for the PR
	prTitle, err := createPRBranch()
	if err != nil {
		return fmt.Errorf("failed to create PR branch: %w", err)
	}
	// Extract notes before any operations
	notes, ok := git.GetNotes()
	if !ok {
		return errors.New("Warning: No notes found in dev branch")
	}

	// Ask for confirmation before creating the PR
	needDraft := false
	if cmd.Flag(prdraftParamFull).Value.String() == trueStr {
		needDraft = true
	}

	prMsg := strings.ReplaceAll(prConfirm, "$prname", prTitle)
	fmt.Print(prMsg)
	_, _ = fmt.Scanln(&response)

	switch response {
	case pushYes:
		err := git.MakePR(prTitle, notes, needDraft)
		if err != nil {
			return fmt.Errorf("failed to create PR: %w", err)
		}
	default:
		fmt.Print(pushFail)
	}

	return nil
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

// Helper function to convert issue URL to repository URL
func convertIssuesURLToRepoURL(issueURL string) (string, error) {
	// Remove trailing issue number
	issuesParts := strings.Split(issueURL, "/issues/")
	if len(issuesParts) < 2 {
		return "", fmt.Errorf("invalid GitHub issue URL format: %s", issueURL)
	}

	repoURL := issuesParts[0]
	return repoURL, nil
}

// createPRBranch creates a new branch for the pull request and checks out on it.
// Returns:
// - title of the pull request
// - error if any operation fails
func createPRBranch() (string, error) {
	// Save current branch name (dev branch)
	devBranchName := git.GetCurrentBranchName()

	// Extract notes before any operations
	notes, ok := git.GetNotes()
	if !ok {
		return "", errors.New("Warning: No notes found in dev branch")
	}

	// checking out on main branch
	mainBranch := git.GetMainBranch()
	_, _, err := new(exec.PipedExec).
		Command("git", "checkout", mainBranch).
		Command("git", "pull", "origin", mainBranch).
		RunToStrings()
	if err != nil {
		return "", err
	}

	// building pr branch name
	prBranchName := strings.TrimSuffix(devBranchName, "-dev") + "-pr"

	// creating new branch for PR from updated main
	_, _, err = new(exec.PipedExec).Command("git", "checkout", "-b", prBranchName).RunToStrings()
	if err != nil {
		return "", err
	}

	// Squash merge dev branch commits
	_, _, err = new(exec.PipedExec).Command("git", "merge", "--squash", devBranchName).RunToStrings()
	if err != nil {
		return "", err
	}

	// get json notes from dev branch
	newNotes, err := notesPkg.Deserialize(notes)
	if err != nil {
		return "", fmt.Errorf("Error deserializing notes: %w", err)
	}

	prTitle := newNotes.GithubIssueURL
	// get issue description from notes for commit message
	issueDescription, err := getIssueDescription(prTitle)
	if err != nil {
		return "", fmt.Errorf("Error retrieving issue description: %w", err)
	}

	// Create commit with the squashed changes
	_, _, err = new(exec.PipedExec).Command("git", "commit", "-m", issueDescription).RunToStrings()
	if err != nil {
		return "", err
	}

	// Add empty commit to create commit object and link notes to it
	err = new(exec.PipedExec).Command("git", "commit", "--allow-empty", "-m", git.MsgCommitForNotes).
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return "", err
	}

	newNotes.BranchType = int(types.BranchTypePr)
	// Add empty commit to create commit object and link notes to it
	git.AddNotes([]string{newNotes.String()})

	// Push notes to origin
	err = new(exec.PipedExec).Command("git", "push", "origin", "ref/notes/*").Run(os.Stdout, os.Stdout)
	// Push PR branch to origin
	_, _, err = new(exec.PipedExec).Command("git", "push", "-u", "origin", prBranchName).RunToStrings()
	if err != nil {
		return "", err
	}

	// Delete dev branch locally and remotely
	_, _, err = new(exec.PipedExec).Command("git", "branch", "-D", devBranchName).RunToStrings()
	if err != nil {
		return "", err
	}

	_, _, err = new(exec.PipedExec).Command("git", "push", "origin", "--delete", devBranchName).RunToStrings()
	if err != nil {
		return "", err
	}

	return prTitle, nil
}

// doesPrExist checks if a pull request exists for the current branch.
func doesPrExist() bool {
	branchName := git.GetCurrentBranchName()
	stdout, _, err := new(exec.PipedExec).
		Command("gh", "pr", "list", "--head", branchName).
		RunToStrings()
	git.ExitIfError(err)

	if strings.Contains(stdout, "no pull requests match your search") {
		return false
	}

	// Otherwise, assume PR exists
	if stdout == "" {
		// safety check: gh may sometimes return nothing at all
		return false
	}

	return true
}

func issueNote(notes []string) bool {
	if len(notes) == 0 {
		return false
	}
	for _, s := range notes {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			if strings.Contains(s, git.IssueSign) {
				return true
			}
		}
	}
	return false
}

func GetCommentForPR(notes []string) (strnote string) {
	strnote = ""
	if len(notes) == 0 {
		return strnote
	}
	for _, note := range notes {
		note = strings.TrimSpace(note)
		if (strings.Contains(note, "https://") && strings.Contains(note, "/issues/")) || !strings.Contains(note, "https://") {
			if len(note) > 0 {
				strnote = strnote + oneSpace + note
			}
		}
	}
	return strings.TrimSpace(strnote)
}

func getIssueNumFromNotes(notes []string) (string, bool) {
	if len(notes) == 0 {
		return "", false
	}
	for _, s := range notes {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			if strings.Contains(s, git.IssueSign) {
				arr := strings.Split(s, oneSpace)
				if len(arr) > 1 {
					num := arr[1]
					if strings.Contains(num, "#") {
						num = strings.ReplaceAll(num, "#", "")
						return num, true
					}
				}
			}
		}
	}
	return "", false
}
