package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/qs/git"
	"github.com/untillpro/qs/internal/commands/helper"
	"github.com/untillpro/qs/internal/types"
)

func Pr(cmd *cobra.Command, args []string) {
	globalConfig()
	git.CheckIfGitRepo()

	if !helper.CheckQSver() {
		return
	}
	if !helper.CheckGH() {
		return
	}
	var prurl string
	bDirectPR := true
	if len(args) > 0 {
		if args[0] != prMergeParam {
			fmt.Println(errMsgPRUnkown)
			return
		}
		if len(args) > 1 {
			prurl = args[1]
		}
		bDirectPR = false
	}

	// find out type of the branch
	branchType := git.GetBranchType()
	if branchType != types.BranchTypeDev {
		_, _ = fmt.Fprintln(os.Stderr, "You must be on dev branch")
		os.Exit(1)
	}

	// PR is not created yet
	prExists := doesPrExist()
	if prExists {
		_, _ = fmt.Fprintln(os.Stderr, "Pull request already exists for this branch")
		os.Exit(1)
	}

	parentrepo := git.GetParentRepoName()
	if len(parentrepo) == 0 {
		fmt.Println("You are in trunk. PR is only allowed from forked branch.")
		os.Exit(0)
	}
	curBranch := git.GetCurrentBranchName()
	isMainBranch := (curBranch == "main") || (curBranch == "master")
	if isMainBranch {
		fmt.Printf("\nUnable to create a pull request on branch '%s'. Use 'qs dev <branch_name>.\n", curBranch)
		os.Exit(0)
	}

	var response string
	if git.UpstreamNotExist() {
		fmt.Print("Upstream not found.\nRepository " + parentrepo + " will be added as upstream. Agree[y/n]?")
		fmt.Scanln(&response)
		if response != pushYes {
			fmt.Print(pushFail)
			return
		}
		response = ""
		git.MakeUpstreamForBranch(parentrepo)
	}

	if git.PRAhead() {
		fmt.Print("This branch is out-of-date. Merge automatically[y/n]?")
		fmt.Scanln(&response)
		if response != pushYes {
			fmt.Print(pushFail)
			return
		}
		response = ""
		git.MergeFromUpstreamRebase()
	}

	var err error
	if bDirectPR {

		if _, ok := git.ChangedFilesExist(); ok {
			fmt.Println(errMsgModFiles)
			return
		}

		notes, ok := git.GetNotes()
		issueNum := ""
		issueok := false
		if !ok || issueNote(notes) {
			issueNum, issueok = getIssueNumFromNotes(notes)
			if !issueok {
				issueNum, issueok = git.GetIssueNumFromBranchName(parentrepo, curBranch)
			}
		}
		if !ok && issueok {
			// Try to get github issue name by branch name
			notes = git.GetIssuePRTitle(issueNum, parentrepo)
			ok = true
		}
		if !ok {
			// Ask PR title
			fmt.Println(errMsgPRNotesNotFound)
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()

			prnotes := scanner.Text()
			prnotes = strings.TrimSpace(prnotes)
			notes = append(notes, prnotes)
		}
		strnotes := git.GetBodyFromNotes(notes)
		if len(strings.TrimSpace(strnotes)) > 0 {
			strnotes = strings.ReplaceAll(strnotes, "Resolves ", "")
		} else {
			strnotes = GetCommentForPR(notes)
		}
		if len(strnotes) > 0 {
			needDraft := false
			if cmd.Flag(prdraftParamFull).Value.String() == trueStr {
				needDraft = true
			}
			prMsg := strings.ReplaceAll(prConfirm, "$prname", strnotes)
			fmt.Print(prMsg)
			fmt.Scanln(&response)
			switch response {
			case pushYes:
				err = git.MakePR(strnotes, notes, needDraft)
			default:
				fmt.Print(pushFail)
			}
			response = ""
		}
	} else {
		err = git.MakePRMerge(prurl)
	}
	if err != nil {
		fmt.Println(err)
		return
	}
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
