package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/untillpro/goutils/logger"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/qs/gitcmds"
	"github.com/untillpro/qs/internal/commands/helper"
	notesPkg "github.com/untillpro/qs/internal/notes"
	"github.com/untillpro/qs/internal/types"
)

func Pr(cmd *cobra.Command, wd string) error {
	globalConfig()
	if _, err := gitcmds.CheckIfGitRepo(wd); err != nil {
		return err
	}

	if !helper.CheckQsVer() {
		return fmt.Errorf("qs version check failed")
	}
	if !helper.CheckGH() {
		return fmt.Errorf("GitHub CLI check failed")
	}

	// find out type of the branch
	branchType := gitcmds.GetBranchType(wd)
	if branchType != types.BranchTypeDev {
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

	parentrepo, err := gitcmds.GetParentRepoName(wd)
	if err != nil {
		return err
	}
	if len(parentrepo) == 0 {
		return errors.New("You are in trunk. PR is only allowed from forked branch.")
	}
	curBranch := gitcmds.GetCurrentBranchName(wd)
	isMainBranch := (curBranch == "main") || (curBranch == "master")
	if isMainBranch {
		return fmt.Errorf("Unable to create a pull request on branch '%s'. Use 'qs dev <branch_name>.", curBranch)
	}

	var response string
	if gitcmds.UpstreamNotExist(wd) {
		fmt.Print("Upstream not found.\nRepository " + parentrepo + " will be added as upstream. Agree[y/n]?")
		_, _ = fmt.Scanln(&response)
		if response != pushYes {
			fmt.Print(pushFail)
			return nil
		}
		response = ""
		if err := gitcmds.MakeUpstreamForBranch(wd, parentrepo); err != nil {
			return fmt.Errorf("failed to set upstream: %w", err)
		}
	}

	//ok, err := gitcmds.PRAhead(wd)
	//if err != nil {
	//	return err
	//}
	//if ok {
	//	fmt.Print("This branch is out-of-date. Merge automatically[y/n]?")
	//	_, _ = fmt.Scanln(&response)
	//	if response != pushYes {
	//		fmt.Print(pushFail)
	//		return nil
	//	}
	//	response = ""
	//	if err := gitcmds.MergeFromUpstreamRebase(wd); err != nil {
	//		return err
	//	}
	//}
	//
	// Check if there are any modified files in the current branch
	if _, ok, err := gitcmds.ChangedFilesExist(wd); ok || err != nil {
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
	notes, ok := gitcmds.GetNotes(wd)
	if !ok {
		return errors.New("Error: No notes found in dev branch")
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
		stdout, stderr, err := gitcmds.MakePR(wd, prTitle, notes, needDraft)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stdout, stdout)
			_, _ = fmt.Fprintln(os.Stderr, stderr)

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
func createPRBranch(wd string) (string, error) {
	// Save current branch name (dev branch)
	devBranchName := gitcmds.GetCurrentBranchName(wd)
	prBranchName := strings.TrimSuffix(devBranchName, "-dev") + "-pr"

	upstreamRemote := "upstream"
	exists, err := gitcmds.HasRemote(wd, upstreamRemote)
	if err != nil {
		return "", err
	}
	if !exists {
		upstreamRemote = "origin"
	}
	upstreamMain := upstreamRemote + "/main"

	// Step 1: Fetch latest upstream
	_, _, err = new(exec.PipedExec).
		Command("git", "fetch", upstreamRemote).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return "", err
	}

	// Step 2: Checkout dev branch

	// extract notes from dev branch before any operations
	notes, ok := gitcmds.GetNotes(wd)
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

	// Step 3: Merge upstream/main into dev branch (non-rebase)
	_, _, err = new(exec.PipedExec).
		Command("git", "merge", upstreamMain).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return "", err
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

	// get json notes from dev branch
	newNotes, err := notesPkg.Deserialize(notes)
	if err != nil {
		return "", fmt.Errorf("Error deserializing notes: %w", err)
	}

	issueDescription, err := getIssueDescription(newNotes.GithubIssueURL)
	if err != nil {
		return "", fmt.Errorf("Error retrieving issue description: %w", err)
	}

	newNotes.BranchType = int(types.BranchTypePr)
	// Step 7: Commit the squashed changes
	stdout, stderr, err := new(exec.PipedExec).
		Command("git", "commit", "-m", issueDescription).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stdout)
		logger.Verbose(stderr)

		return "", err
	}

	// Add empty commit to create commit object and link notes to it
	if err := gitcmds.AddNotes(wd, []string{newNotes.String()}); err != nil {
		return "", err
	}

	// Step 8: Push notes to origin
	err = new(exec.PipedExec).
		Command("git", "push", "origin", "ref/notes/*").
		WorkingDir(wd).
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return "", err
	}

	time.Sleep(2 * time.Second)

	// Step 9: Push PR branch to origin
	_, _, err = new(exec.PipedExec).
		Command("git", "push", "-u", "origin", prBranchName).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return "", err
	}

	time.Sleep(2 * time.Second)

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

// doesPrExist checks if a pull request exists for the current branch.
func doesPrExist(wd string) (bool, error) {
	branchName := gitcmds.GetCurrentBranchName(wd)
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

func issueNote(notes []string) bool {
	if len(notes) == 0 {
		return false
	}
	for _, s := range notes {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			if strings.Contains(s, gitcmds.IssueSign) {
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
			if strings.Contains(s, gitcmds.IssueSign) {
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
