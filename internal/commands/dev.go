package commands

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/fatih/color"
	gitPkg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/gitcmds"
	contextPkg "github.com/untillpro/qs/internal/context"
	"github.com/untillpro/qs/internal/helper"
	"github.com/untillpro/qs/internal/jira"
)

func Dev(cmd *cobra.Command, wd string, args []string) error {
	if _, err := gitcmds.CheckIfGitRepo(wd); err != nil {
		return err
	}

	parentRepo, err := gitcmds.GetParentRepoName(wd)
	if err != nil {
		return err
	}

	// qs dev -d is running
	if cmd.Flag(devDelParamFull).Value.String() == trueStr {
		return deleteBranches(wd, parentRepo)
	}
	// qs dev is running
	var branch string
	var notes []string
	var response string

	if len(args) == 0 {
		clipargs := strings.TrimSpace(getArgStringFromClipboard(cmd.Context()))
		args = append(args, clipargs)
	}

	noForkAllowed := (cmd.Flag(noForkParamFull).Value.String() == trueStr)
	if !noForkAllowed {
		if len(parentRepo) == 0 { // main repository, not forked
			repo, org, err := gitcmds.GetRepoAndOrgName(wd)
			if err != nil {
				return err
			}

			return fmt.Errorf("You are in %s/%s repo\nExecute 'qs fork' first\n", org, repo)
		}
	}

	curBranch, isMain, err := gitcmds.IamInMainBranch(wd)
	if err != nil {
		return err
	}
	if !isMain {
		fmt.Println("--------------------------------------------------------")
		fmt.Println("You are in")
		repo, org, err := gitcmds.GetRepoAndOrgName(wd)
		if err != nil {
			return err
		}

		color.New(color.FgHiCyan).Println(org + "/" + repo + "/" + curBranch)

		return fmt.Errorf("Switch to main branch before running 'qs dev'. You are in %s branch ", curBranch)
	}

	// Stash current changes if needed
	stashedUncommittedChanges := false
	if ok, err := gitcmds.HaveUncommittedChanges(wd); ok {
		if err != nil {
			return err
		}

		if err := gitcmds.Stash(wd); err != nil {
			return fmt.Errorf("error stashing changes: %w", err)
		}
		stashedUncommittedChanges = true
	}

	// sync local MainBranch to ensure it's up to date with origin and upstream remotes
	if err := gitcmds.SyncMainBranch(wd); err != nil {
		return err
	}

	issueNum, githubIssueURL, ok, err := argContainsGithubIssueLink(wd, args...)
	if err != nil {
		return err
	}

	checkRemoteBranchExistence := true
	if ok { // github issue
		fmt.Print("Dev branch for issue #" + strconv.Itoa(issueNum) + " will be created. Agree?(y/n)")
		_, _ = fmt.Scanln(&response)
		if response == pushYes {
			branch, notes, err = gitcmds.DevIssue(wd, parentRepo, githubIssueURL, issueNum, args...)
			if err != nil {
				return err
			}
			checkRemoteBranchExistence = false // no need to check remote branch existence for issue branch
		}
	} else { // PK topic or Jira issue
		if _, ok := jira.GetJiraTicketIDFromArgs(args...); ok { // Jira issue
			branch, notes, err = jira.GetJiraBranchName(args...)
		} else {
			branch, notes, err = helper.GetBranchName(false, args...)
			branch += "-dev" // Add suffix "-dev" for a dev branch
		}
		if err != nil {
			return err
		}

		devMsg := strings.ReplaceAll(devConfirm, "$reponame", branch)
		fmt.Print(devMsg)
		_, _ = fmt.Scanln(&response)
	}

	// put branch name to command context
	cmd.SetContext(context.WithValue(cmd.Context(), contextPkg.CtxKeyDevBranchName, branch))

	exists, err := branchExists(wd, branch)
	if err != nil {
		return fmt.Errorf("error checking branch existence: %w", err)
	}
	if exists {
		return fmt.Errorf("dev branch '%s' already exists", branch)
	}

	switch response {
	case pushYes:
		// Remote developer branch, linked to issue is created
		var response string
		if len(parentRepo) > 0 {
			if gitcmds.UpstreamNotExist(wd) {
				fmt.Print("Upstream not found.\nRepository " + parentRepo + " will be added as upstream. Agree[y/n]?")
				_, _ = fmt.Scanln(&response)
				if response != pushYes {
					fmt.Print(msgOkSeeYou)
					return nil
				}
				response = ""
				if err := gitcmds.MakeUpstreamForBranch(wd, parentRepo); err != nil {
					return err
				}
			}
		}

		if err := gitcmds.Dev(wd, branch, notes, checkRemoteBranchExistence); err != nil {
			return err
		}
	default:
		fmt.Print(msgOkSeeYou)

		return nil
	}

	// Create pre-commit hook to control committing file size
	if err := setPreCommitHook(wd); err != nil {
		logger.Verbose("Error setting pre-commit hook:", err)
	}
	// Unstash changes
	if stashedUncommittedChanges {
		if err := gitcmds.Unstash(wd); err != nil {
			return fmt.Errorf("error unstashing changes: %w", err)
		}
	}

	return nil
}

// branchExists checks if a branch with the given name already exists in the current git repository.
func branchExists(wd, branchName string) (bool, error) {
	repo, err := gitPkg.PlainOpen(wd)
	if err != nil {
		return false, fmt.Errorf("failed to open cloned repository: %w", err)
	}

	branches, err := repo.Branches()
	if err != nil {
		return false, fmt.Errorf("failed to get branches: %w", err)
	}

	// Find development branch name that starts with the issue ID
	exists := false
	_ = branches.ForEach(func(ref *plumbing.Reference) error {
		nextBranchName := ref.Name().Short()
		if nextBranchName == branchName {
			exists = true

			return nil
		}

		return nil
	})

	return exists, nil
}

// getArgStringFromClipboard retrieves a string from the clipboard, or uses the context value if available.
func getArgStringFromClipboard(ctx context.Context) string {
	var err error
	// context value is first
	arg, ok := ctx.Value(contextPkg.CtxKeyClipboard).(string)
	if !ok || len(arg) == 0 {
		arg, err = clipboard.ReadAll()
		if err != nil {
			return ""
		}
	}

	args := strings.Split(arg, "\n")
	var newArg string
	for _, str := range args {
		newArg += str
		newArg += oneSpace
	}

	return newArg
}

func setPreCommitHook(wd string) error {
	if ok, err := gitcmds.LocalPreCommitHookExist(wd); ok || err != nil {
		return err
	}

	return gitcmds.SetLocalPreCommitHook(wd)
}

func argContainsGithubIssueLink(wd string, args ...string) (issueNum int, issueURL string, ok bool, err error) {
	ok = false
	if len(args) != 1 {
		return
	}
	url := args[0]
	if strings.Contains(url, "/issues") {
		if err := checkIssueLink(wd, url); err != nil {
			return 0, "", false, fmt.Errorf("Invalid GitHub issue link: %w", err)
		}
		segments := strings.Split(url, "/")
		strIssueNum := segments[len(segments)-1]
		i, err := strconv.Atoi(strIssueNum)
		if err != nil {
			return 0, "", false, fmt.Errorf("failed to convert issue number from string to int: %w", err)
		}

		return i, url, true, nil
	}

	return 0, "", false, nil
}

func checkIssueLink(wd, issueURL string) error {
	// This function checks if the provided issueURL is a valid GitHub issue link via `gh issue view`.
	cmd := exec.Command("gh", "issue", "view", issueURL)
	cmd.Dir = wd
	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("failed to check issue link: %v", err)
	}

	return nil
}

func deleteBranches(wd, parentRepo string) error {
	// Step 1: qs d
	if err := gitcmds.Download(wd); err != nil {
		return err
	}

	mainBranch, err := gitcmds.GetMainBranch(wd)
	if err != nil {
		return err
	}

	// Step 2: Checkout Main
	if err := gitcmds.CheckoutOnBranch(wd, mainBranch); err != nil {
		return err
	}

	// Step 3: foreach branch that have origin remote tracking branch `git branch -vv | awk '$3 ~ /\[origin.*\]/ {print $1}'`
	branchesToAnalyze, err := gitcmds.GetBranchesWithRemoteTracking(wd, "origin")
	if err != nil {
		return err
	}

	// Iterate through branches
	for _, branch := range branchesToAnalyze {
		// Step 3.n: if pr is merged, then all related branches must be deleted
		mergedPrExists, _, _, _, err := gitcmds.DoesPrExist(wd, parentRepo, branch, gitcmds.PRStateMerged)
		if err != nil {
			return err
		}
		// if pr is not merged yet then branch must live
		if !mergedPrExists {
			continue
		}

		if err := gitcmds.RemoveBranch(wd, branch); err != nil {
			return err
		}
	}

	return nil
}
