package commands

import (
	"bytes"
	"encoding/json"
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
	"github.com/untillpro/qs/git"
	"github.com/untillpro/qs/internal/commands/helper"
	"github.com/untillpro/qs/internal/notes"
	"github.com/untillpro/qs/internal/types"
)

func Dev(cmd *cobra.Command, args []string) {
	globalConfig()
	git.CheckIfGitRepo()
	// TODO: uncomment before PR
	//if !helper.CheckQSver() {
	//	return
	//}
	if !helper.CheckGH() {
		return
	}

	// qs dev -d is running
	if cmd.Flag(devDelParamFull).Value.String() == trueStr {
		deleteBranches()
		return
	}
	// qs dev is running
	var branch string
	var notes []string
	var response string

	if len(args) == 0 {
		clipargs := strings.TrimSpace(getArgStringFromClipboard())
		args = append(args, clipargs)
	}
	remoteURL := git.GetRemoteUpstreamURL()
	noForkAllowed := (cmd.Flag(noForkParamFull).Value.String() == trueStr)
	if !noForkAllowed {
		parentrepo := git.GetParentRepoName()
		if len(parentrepo) == 0 { // main repository, not forked
			repo, org := git.GetRepoAndOrgName()
			fmt.Printf("You are in %s/%s repo\nExecute 'qs fork' first\n", org, repo)
			return
		}
	}
	curBranch, isMain := git.IamInMainBranch()
	if !isMain {
		fmt.Println("--------------------------------------------------------")
		fmt.Println("You are in")
		repo, org := git.GetRepoAndOrgName()
		color.New(color.FgHiCyan).Println(org + "/" + repo + "/" + curBranch)
		fmt.Println("Switch to main branch before running 'qs dev'")
		return
	}

	// check if there are uncommitted changes and stash them
	stashedUncommittedChanges := false
	if git.HaveUncommittedChanges() {
		if err := git.Stash(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "error stashing changes:", err)
			os.Exit(1)

			return
		}
		stashedUncommittedChanges = true
	}

	issueNum, githubIssueURL, ok := argContainsGithubIssueLink(args...)
	if ok { // github issue
		fmt.Print("Dev branch for issue #" + strconv.Itoa(issueNum) + " will be created. Agree?(y/n)")
		fmt.Scanln(&response)
		if response == pushYes {
			// Remote developer branch, linked to issue is created
			branch, notes = git.DevIssue(githubIssueURL, issueNum, args...)
		}
	} else { // PK topic or Jira issue
		if _, ok := GetJiraTicketIDFromArgs(args...); ok { // Jira issue
			branch, notes = getJiraBranchName(args...)
		} else {
			branch, notes = getBranchName(false, args...)
		}
		devMsg := strings.ReplaceAll(devConfirm, "$reponame", branch)
		fmt.Print(devMsg)
		fmt.Scanln(&response)
	}

	// Add suffix `-dev` for a dev branch
	branch = branch + "-dev"

	exists, err := branchExists(branch)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error checking branch existence:", err)
		os.Exit(1)

		return
	}
	if exists {
		_, _ = fmt.Fprintf(os.Stderr, "dev branch '%s' already exists", branch)
		os.Exit(1)

		return
	}

	switch response {
	case pushYes:
		// Remote developer branch, linked to issue is created
		var response string
		parentrepo := git.GetParentRepoName()
		if len(parentrepo) > 0 {
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
		}
		if len(remoteURL) == 0 {
			git.Dev(branch, notes, false)
		} else {
			git.Dev(branch, notes, true)
		}
	default:
		fmt.Print(pushFail)

		return
	}

	// Create pre-commit hook to control committing file size
	setPreCommitHook()
	// Unstash uncommited changes If needed
	if stashedUncommittedChanges {
		if err := git.Unstash(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "error unstashing changes:", err)
			os.Exit(1)

			return
		}
	}
}

// branchExists checks if a branch with the given name already exists in the current git repository.
func branchExists(branchName string) (bool, error) {
	wd, err := os.Getwd()
	if err != nil {
		return false, fmt.Errorf("error getting current working directory: %s", err)
	}

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

func getArgStringFromClipboard() string {
	arg, err := clipboard.ReadAll()
	if err != nil {
		return ""
	}
	args := strings.Split(arg, "\n")
	var newarg string
	for _, str := range args {
		newarg += str
		newarg += oneSpace
	}
	return newarg
}

func setPreCommitHook() {
	var response string
	if git.LocalPreCommitHookExist() {
		return
	}

	fmt.Print("\nGit pre-commit hook, preventing commit large files does not exist.\nDo you want to set hook(y/n)?")
	fmt.Scanln(&response)
	switch response {
	case pushYes:
		git.SetLocalPreCommitHook()
	default:
		return
	}
}

func getBranchName(ignoreEmptyArg bool, args ...string) (branch string, comments []string) {

	args = clearEmptyArgs(args)
	if len(args) == 0 {
		if ignoreEmptyArg {
			return "", []string{}
		}
		fmt.Println("Need branch name for dev")
		os.Exit(1)
	}

	newargs := splitQuotedArgs(args...)
	comments = newargs
	for i, arg := range newargs {
		arg = strings.TrimSpace(arg)
		if i == 0 {
			branch = arg
			continue
		}
		if i == len(newargs)-1 {
			// Retrieve taskID from url and add it first to branch name
			url := arg
			topicid := getTaskIDFromURL(url)
			if topicid == arg {
				branch = branch + msymbol + topicid
			} else {
				branch = topicid + msymbol + branch
			}
			break
		}
		branch = branch + "-" + arg
	}
	branch = cleanArgfromSpecSymbols(branch)
	return branch, comments
}

func argContainsGithubIssueLink(args ...string) (issueNum int, issueURL string, ok bool) {
	ok = false
	if len(args) != 1 {
		return
	}
	url := args[0]
	if strings.Contains(url, "/issues") {
		if err := checkIssueLink(url); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "Invalid GitHub issue link:", err)
			os.Exit(1)

			return
		}
		segments := strings.Split(url, "/")
		strIssueNum := segments[len(segments)-1]
		i, err := strconv.Atoi(strIssueNum)
		if err != nil {
			return
		}
		return i, url, true
	}

	return
}

func checkIssueLink(issueURL string) error {
	// This function checks if the provided issueURL is a valid GitHub issue link via `gh issue view`.
	cmd := exec.Command("gh", "issue", "view", issueURL)
	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("failed to check issue link: %v", err)
	}

	return nil
}

// getJiraBranchName generates a branch name based on a JIRA issue URL in the arguments.
// If a JIRA URL is found, it generates a branch name in the format "<ISSUE-KEY>-<cleaned-description>".
// Additionally, it generates comments in the format "[<ISSUE-KEY>] <original-line>".
func getJiraBranchName(args ...string) (branch string, comments []string) {
	re := regexp.MustCompile(`https://([a-zA-Z0-9-]+)\.atlassian\.net/browse/([A-Z]+-[A-Z0-9-]+)`)
	for _, arg := range args {
		if matches := re.FindStringSubmatch(arg); matches != nil {
			issueKey := matches[2] // Extract the JIRA issue key (e.g., "AIR-270")

			var brname string
			issuename := getJiraIssueNameByNumber(issueKey)
			if issuename == "" {
				branch, _ = getBranchName(false, args...)
			} else {
				jiraTicketURL := matches[0] // Full JIRA ticket URL
				// Prepare new notes
				newNotes := notes.Serialize("", jiraTicketURL, types.BranchTypeDev)
				comments = append(comments, newNotes)
				brname, _ = getBranchName(false, issuename)
				branch = issueKey + "-" + brname
			}
			comments = append(comments, "["+issueKey+"] "+issuename)
		}
	}
	comments = append(comments, args...)
	return branch, comments
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

func cleanArgfromSpecSymbols(arg string) string {
	var symbol string

	arg = strings.ReplaceAll(arg, "https://", "")
	replaceToMinus := []string{oneSpace, ",", ";", ".", ":", "?", "/", "!"}
	for _, symbol = range replaceToMinus {
		arg = strings.ReplaceAll(arg, symbol, "-")
	}
	replaceToNone := []string{"&", "$", "@", "%", "\\", "(", ")", "{", "}", "[", "]", "<", ">", "'", "\""}
	for _, symbol = range replaceToNone {
		arg = strings.ReplaceAll(arg, symbol, "")
	}
	for string(arg[0]) == msymbol {
		arg = arg[1:]
	}

	arg = deleteDupMinus(arg)
	if len(arg) > maxDevBranchName {
		arg = arg[:maxDevBranchName]
	}
	for string(arg[len(arg)-1]) == msymbol {
		arg = arg[:len(arg)-1]
	}
	return arg
}

func deleteDupMinus(str string) string {
	var buf bytes.Buffer
	var pc rune
	for _, c := range str {
		if pc == c && string(c) == msymbol {
			continue
		}
		pc = c
		buf.WriteRune(c)
	}
	return buf.String()
}

func getJiraIssueNameByNumber(issueNum string) (name string) {
	// Validate the issue key
	if issueNum == "" {
		fmt.Println("Error: Issue key is required.")
		return ""
	}

	// Retrieve API token and email from environment variables
	apiToken := os.Getenv("JIRA_API_TOKEN")
	if apiToken == "" {
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println("Error: JIRA API token not found. Please set environment variable JIRA_API_TOKEN.")
		fmt.Println("            Jira API token can generate on this page:")
		fmt.Println("          https://id.atlassian.com/manage-profile/security/api-tokens           ")
		fmt.Println("--------------------------------------------------------------------------------")
		return ""
	}
	var email string
	email = os.Getenv("JIRA_EMAIL")
	if email == "" {
		email = git.GetUserEmail() // Replace with your email
	}
	if email == "" {
		fmt.Println("Error: Please export JIRA_EMAIL.")
		return ""
	}
	fmt.Println("User email: ", email)
	jiraDomain := "https://untill.atlassian.net"

	// Build the request URL
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", jiraDomain, issueNum)

	// Create HTTP client and request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}
	req.SetBasicAuth(email, apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var result struct {
		Fields struct {
			Summary string `json:"summary"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("Error parsing JSON response:", err)
		return ""
	}

	// Check if the summary field exists
	if result.Fields.Summary == "" {
		return ""
	}

	return result.Fields.Summary
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

func globalConfig() {
	if verbose {
		logger.SetLogLevel(logger.LogLevelVerbose)
	} else {
		logger.SetLogLevel(logger.LogLevelInfo)
	}
}

func deleteBranches() {
	git.PullUpstream()
	lst, err := git.GetMergedBranchList()
	if err != nil {
		fmt.Println(err)
		return
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
		fmt.Scanln(&response)
		switch response {
		case pushYes:
			git.DeleteBranchesRemote(lst)
		default:
			fmt.Print(pushFail)
		}
	}
	git.PullUpstream()

	fmt.Print("\nChecking if unused local branches exist...")
	var strs = git.GetGoneBranchesLocal()

	var strFin []string

	for _, str := range *strs {
		if (strings.TrimSpace(str) != "") && (strings.TrimSpace(str) != "*") {
			strFin = append(strFin, str)
		}
	}

	if len(strFin) == 0 {
		fmt.Println(delLocalBranchNothing)
		return
	}

	fmt.Print(devider)

	for _, str := range strFin {
		fmt.Print("\n" + str)
	}

	fmt.Print(devider)
	fmt.Print(delLocalBranchConfirm)
	fmt.Scanln(&response)
	switch response {
	case pushYes:

		git.DeleteBranchesLocal(strs)

	default:

		fmt.Print(pushFail)
	}

}
