package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
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
	"github.com/untillpro/qs/internal/notes"
	notesPkg "github.com/untillpro/qs/internal/notes"
)

func Dev(cmd *cobra.Command, wd string, args []string) error {
	_, err := gitcmds.CheckIfGitRepo(wd)
	if err != nil {
		return err
	}

	// qs dev -d is running
	if cmd.Flag(devDelParamFull).Value.String() == trueStr {
		return deleteBranches(wd)
	}
	// qs dev is running
	var branch string
	var notes []string
	var response string

	if len(args) == 0 {
		clipargs := strings.TrimSpace(getArgStringFromClipboard(cmd.Context()))
		args = append(args, clipargs)
	}

	parentRepo, err := gitcmds.GetParentRepoName(wd)
	if err != nil {
		return err
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
		if _, ok := GetJiraTicketIDFromArgs(args...); ok { // Jira issue
			branch, notes, err = getJiraBranchName(wd, args...)
		} else {
			branch, notes, err = getBranchName(false, args...)
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
					fmt.Print(pushFail)
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
		fmt.Print(pushFail)

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

func getBranchName(ignoreEmptyArg bool, args ...string) (branch string, comments []string, err error) {

	args = clearEmptyArgs(args)
	if len(args) == 0 {
		if ignoreEmptyArg {
			return "", []string{}, nil
		}

		return "", []string{}, errors.New("Need branch name for dev")
	}

	newArgs := splitQuotedArgs(args...)
	comments = make([]string, 0, len(newArgs)+1) // 1 for json notes
	comments = append(comments, newArgs...)
	for i, arg := range newArgs {
		arg = strings.TrimSpace(arg)
		if i == 0 {
			branch = arg
			continue
		}
		if i == len(newArgs)-1 {
			// Retrieve taskID from url and add it first to branch name
			url := arg
			topicID := getTaskIDFromURL(url)
			if topicID == arg {
				branch = branch + msymbol + topicID
			} else {
				branch = topicID + msymbol + branch
			}
			break
		}
		branch = branch + "-" + arg
	}
	branch = helper.CleanArgFromSpecSymbols(branch)
	// Prepare new notes
	notesObj, err := notes.Serialize("", "", notesPkg.BranchTypeDev)
	if err != nil {
		return "", []string{}, err
	}
	comments = append(comments, notesObj)

	return branch, comments, nil
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

// getJiraBranchName generates a branch name based on a JIRA issue URL in the arguments.
// If a JIRA URL is found, it generates a branch name in the format "<ISSUE-KEY>-<cleaned-description>".
// Additionally, it generates comments in the format "[<ISSUE-KEY>] <original-line>".
func getJiraBranchName(wd string, args ...string) (branch string, comments []string, err error) {
	comments = make([]string, 0, len(args)+1) // 1 for json notes
	re := regexp.MustCompile(`https://([a-zA-Z0-9-]+)\.atlassian\.net/browse/([A-Z]+-[A-Z0-9-]+)`)
	for _, arg := range args {
		if matches := re.FindStringSubmatch(arg); matches != nil {
			issueKey := matches[2] // Extract the JIRA issue key (e.g., "AIR-270")

			var brName string
			issueName, err := getJiraIssueNameByNumber(issueKey)
			if err != nil {
				return "", nil, err
			}
			if issueName == "" {
				branch, _, err = getBranchName(false, args...)
				if err != nil {
					return "", nil, err
				}
			} else {
				jiraTicketURL := matches[0] // Full JIRA ticket URL
				// Prepare new notes
				notesObj, err := notes.Serialize("", jiraTicketURL, notesPkg.BranchTypeDev)
				if err != nil {
					return "", nil, err
				}
				comments = append(comments, notesObj)
				brName, _, err = getBranchName(false, issueName)
				if err != nil {
					return "", nil, err
				}
				branch = issueKey + "-" + brName
			}
			comments = append(comments, "["+issueKey+"] "+issueName)
		}
	}
	// Add suffix "-dev" for a dev branch
	branch += "-dev"
	comments = append(comments, args...)

	return branch, comments, nil
}

func clearEmptyArgs(args []string) (newargs []string) {
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if len(arg) > 0 {
			newargs = append(newargs, arg)
		}
	}
	return
}

func splitQuotedArgs(args ...string) []string {
	var newargs []string
	for _, arg := range args {
		subargs := strings.Split(arg, oneSpace)
		if len(subargs) == 0 {
			continue
		}
		for _, a := range subargs {
			if len(a) > 0 {
				newargs = append(newargs, a)
			}
		}
	}
	return newargs
}

func getJiraIssueNameByNumber(issueNum string) (name string, err error) {
	// Validate the issue key
	if issueNum == "" {
		return "", errors.New("Error: Issue key is required.")
	}

	// Retrieve API token and email from environment variables
	apiToken := os.Getenv("JIRA_API_TOKEN")
	if apiToken == "" {
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println("Error: JIRA API token not found. Please set environment variable JIRA_API_TOKEN.")
		fmt.Println("            Jira API token can generate on this page:")
		fmt.Println("          https://id.atlassian.com/manage-profile/security/api-tokens           ")
		fmt.Println("--------------------------------------------------------------------------------")

		return "", errors.New("Error: JIRA API token not found.")
	}
	var email string
	email = os.Getenv("JIRA_EMAIL")
	if email == "" {
		email, err = gitcmds.GetUserEmail() // Replace with your email
	}
	if err != nil {
		return "", err
	}
	if email == "" {
		return "", errors.New("Error: Please export JIRA_EMAIL.")
	}
	fmt.Println("User email: ", email)
	jiraDomain := "https://untill.atlassian.net"

	// Build the request URL
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", jiraDomain, issueNum)

	// Create HTTP client and request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(email, apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", err
	}

	var result struct {
		Fields struct {
			Summary string `json:"summary"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("Error parsing JSON response: %w", err)
	}

	// Check if the summary field exists
	if result.Fields.Summary == "" {
		return "", nil
	}

	return result.Fields.Summary, nil
}

func getTaskIDFromURL(url string) string {
	var entry string
	str := strings.Split(url, "/")
	if len(str) > 0 {
		entry = str[len(str)-1]
	}
	entry = strings.ReplaceAll(entry, "#", "")
	entry = strings.ReplaceAll(entry, "!", "")
	return strings.TrimSpace(entry)
}

// GetJiraTicketIDFromArgs retrieves a JIRA ticket ID from the provided arguments.
// Parameters:
// - args: A variable number of string arguments that may contain JIRA issue URLs.
// Returns:
// - jiraTicketID: The JIRA ticket ID if found.
// - ok: A boolean indicating whether a JIRA ticket ID was found.
func GetJiraTicketIDFromArgs(args ...string) (jiraTicketID string, ok bool) {
	// Define a regular expression to match JIRA issue URLs
	// The regex matches URLs like "https://<subdomain>.atlassian.net/browse/<ISSUE-KEY>"
	re := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.atlassian\.net/browse/([A-Z]+-[A-Z0-9-]+)`)

	for _, arg := range args {
		// Check if the argument matches the pattern
		if matches := re.FindStringSubmatch(arg); matches != nil {
			// Return the issue key (group 1) and true
			return matches[1], true
		}
	}

	// No matching argument was found
	return "", false
}

func deleteBranches(wd string) error {
	if err := gitcmds.PullUpstream(wd); err != nil {
		return err
	}

	lst, err := gitcmds.GetMergedBranchList(wd)
	if err != nil {
		return err
	}

	var response string
	if len(lst) == 0 {
		fmt.Println(delBranchNothing)
	} else {
		fmt.Print(devider)
		for _, l := range lst {
			fmt.Print("\n" + l)
		}
		fmt.Print(devider)

		fmt.Print(delBranchConfirm)
		_, _ = fmt.Scanln(&response)
		switch response {
		case pushYes:
			if err := gitcmds.DeleteBranchesRemote(wd, lst); err != nil {
				return err
			}
		default:
			fmt.Print(pushFail)
		}
	}
	if err := gitcmds.PullUpstream(wd); err != nil {
		return err
	}

	fmt.Print("\nChecking if unused local branches exist...")
	strs, err := gitcmds.GetGoneBranchesLocal(wd)
	if err != nil {
		return err
	}

	var strFin []string

	for _, str := range *strs {
		if (strings.TrimSpace(str) != "") && (strings.TrimSpace(str) != "*") {
			strFin = append(strFin, str)
		}
	}

	if len(strFin) == 0 {
		return errors.New(delLocalBranchNothing)
	}

	fmt.Print(devider)

	for _, str := range strFin {
		fmt.Print("\n" + str)
	}

	fmt.Print(devider)
	fmt.Print(delLocalBranchConfirm)
	_, _ = fmt.Scanln(&response)
	switch response {
	case pushYes:
		if err := gitcmds.DeleteBranchesLocal(wd, strs); err != nil {
			return err
		}
	default:
		fmt.Print(pushFail)
	}

	return nil
}
