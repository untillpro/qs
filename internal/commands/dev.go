package commands

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/untillpro/qs/gitcmds"
	"github.com/untillpro/qs/internal/issue"
	"github.com/untillpro/qs/internal/jira"
	"github.com/untillpro/qs/internal/notes"
	"github.com/untillpro/qs/utils"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

func Dev(cmd *cobra.Command, wd string, doDelete bool, ignoreHook bool, args []string) error {
	rc, err := gitcmds.LoadRepoContext(wd)
	if err != nil {
		return err
	}

	// qs dev -d is running
	if doDelete {
		return deleteBranches(wd, rc.ParentRepo)
	}
	// qs dev is running
	var devBranchName string
	var notes []string
	var response string

	if len(args) == 0 {
		clipargs := strings.TrimSpace(getArgStringFromClipboard(cmd.Context()))
		args = append(args, clipargs)
	}

	// Auto-detect workflow mode:
	// - If parentRepo exists OR upstream remote exists -> fork workflow
	// - If no parentRepo AND no upstream remote -> single remote workflow
	// Check if upstream remote exists

	// Only require fork if:
	// 1. No parent repo exists (not a fork), AND
	// 2. Upstream remote exists (indicating fork workflow was intended)
	// This catches the edge case where someone manually added upstream but didn't fork
	if len(rc.ParentRepo) == 0 && rc.UpstreamExists {
		return fmt.Errorf("you are in %s/%s repo with upstream remote but no fork detected\nExecute 'qs fork' first", rc.Org, rc.Repo)
	}

	if !rc.IsMain {
		fmt.Println("--------------------------------------------------------")
		fmt.Println("You are in")
		color.New(color.FgHiCyan).Println(rc.Org + "/" + rc.Repo + "/" + rc.CurrentBranch)

		return fmt.Errorf("switch to main branch before running 'qs dev'. You are in %s branch ", rc.CurrentBranch)
	}

	// Stash current changes if needed
	stashedUncommittedChanges := false
	if rc.HasUncommittedChanges {
		if err := gitcmds.Stash(wd); err != nil {
			return fmt.Errorf("error stashing changes: %w", err)
		}
		stashedUncommittedChanges = true
	}

	// sync local MainBranch to ensure it's up to date with origin and upstream remotes
	if err := gitcmds.SyncMainBranch(wd, rc.MainBranch, rc.UpstreamExists); err != nil {
		return err
	}

	issueInfo, err := issue.ParseIssueFromArgs(args...)
	if err != nil {
		return err
	}

	devBranchName, notes, err = issue.BuildDevBranchName(issueInfo)
	if err != nil {
		if errors.Is(err, jira.ErrJiraIssueNotFoundOrInsufficientPermission) {
			fmt.Print(jira.NotFoundIssueOrInsufficientAccessRightSuggestion)
			return nil
		}
		return err
	}

	exists, err := branchExists(wd, devBranchName)
	if err != nil {
		return fmt.Errorf("error checking branch existence: %w", err)
	}
	if exists {
		return fmt.Errorf("dev branch '%s' already exists", devBranchName)
	}

	cmd.SetContext(context.WithValue(cmd.Context(), utils.CtxKeyDevBranchName, devBranchName))

	fmt.Print("Dev branch '" + devBranchName + "' will be created. Continue(y/n)? ")
	_, _ = fmt.Scanln(&response)

	switch response {
	case pushYes:
		if len(rc.ParentRepo) > 0 && !rc.UpstreamExists {
			var upstreamResponse string
			fmt.Print("Upstream not found.\nRepository " + rc.ParentRepo + " will be added as upstream. Agree[y/n]?")
			_, _ = fmt.Scanln(&upstreamResponse)
			if upstreamResponse != pushYes {
				fmt.Print(msgOkSeeYou)
				return nil
			}
			if err := gitcmds.MakeUpstreamForBranch(wd, rc.ParentRepo); err != nil {
				return err
			}
		}

		if err := gitcmds.CreateDevBranch(wd, devBranchName, rc.MainBranch, notes); err != nil {
			return err
		}

		if issueInfo.Type == issue.GitHub {
			notes, err = gitcmds.LinkBranchToGithubIssue(wd, rc.ParentRepo, issueInfo.Text, issueInfo.ID, devBranchName, args...)
			if err != nil {
				return err
			}
		}
	default:
		fmt.Print(msgOkSeeYou)
		return nil
	}

	// Create pre-commit hook to control committing file size
	if err := setPreCommitHook(wd); err != nil {
		logger.Verbose("Error setting pre-commit hook:", err)
	}

	// Ensure large file hook content is up to date
	if err := gitcmds.EnsureLargeFileHookUpToDate(wd); err != nil {
		logger.Verbose("Error updating large file hook content:", err)
	}
	// Unstash changes
	if stashedUncommittedChanges {
		if err := gitcmds.Unstash(wd); err != nil {
			return fmt.Errorf("error unstashing changes: %w", err)
		}
	}

	return nil
}

func branchExists(wd, branchName string) (bool, error) {
	cmd := exec.Command("git", "branch", "--list", branchName)
	cmd.Dir = wd
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check local branches: %w", err)
	}
	if strings.TrimSpace(string(output)) != "" {
		return true, nil
	}

	cmd = exec.Command("git", "ls-remote", "--heads", "origin", branchName)
	cmd.Dir = wd
	output, err = cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check remote branches: %w", err)
	}
	return strings.TrimSpace(string(output)) != "", nil
}

// getArgStringFromClipboard retrieves a string from the clipboard, or uses the context value if available.
func getArgStringFromClipboard(ctx context.Context) string {
	var err error
	// context value is first
	arg, ok := ctx.Value(utils.CtxKeyClipboard).(string)
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
		newArg += " "
	}

	return newArg
}

func setPreCommitHook(wd string) error {
	if ok, err := gitcmds.LocalPreCommitHookExist(wd); ok || err != nil {
		return err
	}

	return gitcmds.SetLocalPreCommitHook(wd)
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

	branchesToBeDeleted := make([]string, 0, len(branchesToAnalyze))
	// Iterate through branches
	for _, branch := range branchesToAnalyze {
		// Step 3.n: if pr is merged, then all related branches must be deleted
		prInfo, _, _, err := gitcmds.DoesPrExist(wd, parentRepo, branch, gitcmds.PRStateMerged)
		if err != nil {
			return err
		}
		// if pr is not merged yet then branch must live
		if prInfo == nil {
			skipBranch := true
			// if dev branch then check if pull request is merged of the possible related pr branch
			branchType := gitcmds.GetBranchTypeByName(branch)
			if branchType == notes.BranchTypeDev {
				// calculate possible related pr branch name
				// e.g. if branch is "feature-123-dev" then related pr branch
				// is "feature-123-pr"
				prBranchName := strings.TrimSuffix(branch, "-dev") + "-pr"
				// check if pull request is merged of the possible related pr branch
				prInfo, _, _, err := gitcmds.DoesPrExist(wd, parentRepo, prBranchName, gitcmds.PRStateMerged)
				if err != nil {
					return err
				}
				// if pr is merged then remove dev branch
				if prInfo != nil {
					skipBranch = false
				}
			}
			if skipBranch {
				continue
			}
		}

		branchesToBeDeleted = append(branchesToBeDeleted, branch)
	}

	// Step4: show branches to be deleted
	if len(branchesToBeDeleted) > 0 {
		fmt.Println("Branches to be deleted:")
		for _, branch := range branchesToBeDeleted {
			fmt.Println(branch)
		}

		// Step 5: ask for confirmation
		var response string
		fmt.Println()
		fmt.Print("Proceed with deletion? [y/n]?")
		_, _ = fmt.Scanln(&response)
		if response != pushYes {
			fmt.Print(msgOkSeeYou)
			return nil
		}

		// Step 6: deletion branches
		for _, branch := range branchesToBeDeleted {
			if err := gitcmds.RemoveBranch(wd, branch); err != nil {
				return fmt.Errorf("error deleting branch '%s': %w", branch, err)
			}

			fmt.Printf("Branch '%s' deleted successfully.\n", branch)
		}

		return nil
	}

	fmt.Println("No branches to delete.")

	return nil
}
