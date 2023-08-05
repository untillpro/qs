package git

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/untillpro/gochips"
	"github.com/untillpro/qs/utils"
	"github.com/untillpro/qs/vcs"
)

const (
	mimm                  = "-m"
	slash                 = "/"
	git                   = "git"
	push                  = "push"
	fetch                 = "fetch"
	branch                = "branch"
	checkout              = "checkout"
	origin                = "origin"
	httppref              = "https"
	pushYes               = "y"
	nochecksmsg           = "no checks reported"
	msgWaitingPR          = "Waiting PR checks.."
	msgPRCheckNotFoundYet = "..not found yet"
	msgPRCheckNotFound    = "No checks for PR found, merge without checks"
	MsgPreCommitError     = "Attempt to commit too"

	repoNotFound            = "git repo name not found"
	userNotFound            = "git user name not found"
	errAlreadyForkedMsg     = "You are in fork already\nExecute 'qs dev [branch name]' to create dev branch"
	errMsgPRNotesImpossible = "Pull request without comments is impossible."
	errMsgPRMerge           = "URL of PR is needed"
	errMsgPRBadFormat       = "Pull request URL has bad format"
	errTimer40Sec           = "Time out 40 seconds"
	errSomethigWrong        = "Something went wrong"
	errUnknowGHResponse     = "Unkown response from gh"
	PushDefaultMsg          = "misc"

	IssuePRTtilePrefix = "Resolves issue"
)

type gchResponse struct {
	_stdout string
	_stderr string
	_err    error
}

func CheckIfGitRepo() string {
	stdouts, _, err := new(gochips.PipedExec).
		Command("git", "status", "-s").
		RunToStrings()
	if err != nil {
		if strings.Contains(err.Error(), "128") {
			err = errors.New("This is not a git repository.")
		}
	}
	gochips.ExitIfError(err)
	return stdouts
}

// ChangedFilesExist s.e.
func ChangedFilesExist() (uncommitedFiles string, exist bool) {
	files := CheckIfGitRepo()
	uncommitedFiles = strings.TrimSpace(files)
	exist = len(uncommitedFiles) > 0
	return uncommitedFiles, exist
}

// Stash entries exist ?
func stashEntiresExist() bool {
	stdouts, _, err := new(gochips.PipedExec).
		Command(git, "stash", "list").
		RunToStrings()
	gochips.ExitIfError(err)
	stashentires := strings.TrimSpace(stdouts)
	return len(stashentires) > 0
}

// Status shows git repo status
func Status(cfg vcs.CfgStatus) {
	stdout, _, err := new(gochips.PipedExec).
		Command("git", "remote", "-v").
		Command("grep", fetch).
		Command("sed", "s/(fetch)//").
		RunToStrings()
	if err != nil {
		if strings.Contains(err.Error(), "128") {
			err = errors.New("This is not a git repository.")
		}
	}
	gochips.ExitIfError(err)
	fmt.Print(stdout)
	err = new(gochips.PipedExec).
		Command("git", "status", "-s", "-b", "-uall").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
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
func Release() {

	// *************************************************
	gochips.Doing("Pulling")
	err := new(gochips.PipedExec).
		Command("git", "pull").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	// *************************************************
	gochips.Doing("Reading current version")
	currentVersion, err := utils.ReadVersion()
	gochips.ExitIfError(err, "Error reading file 'version'")
	gochips.ExitIfFalse(len(currentVersion.PreRelease) > 0, "pre-release part of version does not exist: "+currentVersion.String())

	// Calculate target version

	targetVersion := currentVersion
	targetVersion.PreRelease = ""

	fmt.Printf("Version %v will be tagged, bumped and pushed, agree? [y]", targetVersion)
	var response string
	fmt.Scanln(&response)
	gochips.ExitIfFalse(response == "y")

	// *************************************************
	gochips.Doing("Updating 'version' file")
	gochips.ExitIfError(targetVersion.Save())

	// *************************************************
	gochips.Doing("Commiting target version")
	{
		params := []string{"commit", "-a", mimm, "#scm-ver " + targetVersion.String()}
		err = new(gochips.PipedExec).
			Command(git, params...).
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
	}

	// *************************************************
	gochips.Doing("Tagging")
	{
		tagName := "v" + targetVersion.String()
		n := time.Now()
		params := []string{"tag", mimm, "Version " + tagName + " of " + n.Format("2006/01/02 15:04:05"), tagName}
		err = new(gochips.PipedExec).
			Command(git, params...).
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
	}

	// *************************************************
	gochips.Doing("Bumping version")
	newVersion := currentVersion
	{
		newVersion.Minor++
		newVersion.PreRelease = "SNAPSHOT"
		gochips.ExitIfError(newVersion.Save())
	}

	// *************************************************
	gochips.Doing("Commiting new version")
	{
		params := []string{"commit", "-a", mimm, "#scm-ver " + newVersion.String()}
		err = new(gochips.PipedExec).
			Command(git, params...).
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
	}

	// *************************************************
	gochips.Doing("Pushing to origin")
	{
		params := []string{push, "--follow-tags", origin}
		err = new(gochips.PipedExec).
			Command(git, params...).
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
	}
}

// Upload upload sources to git repo
func Upload(cfg vcs.CfgUpload) {
	err := new(gochips.PipedExec).
		Command(git, "add", ".").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	params := []string{"commit", "-a"}
	for _, m := range cfg.Message {
		params = append(params, mimm, m)
	}
	_, sterr, err := new(gochips.PipedExec).
		Command(git, params...).
		RunToStrings()
	if strings.Contains(sterr, MsgPreCommitError) {
		var response string
		fmt.Println("")
		fmt.Println(strings.TrimSpace(sterr))
		fmt.Print("Do you want to commit anyway(y/n)?")
		fmt.Scanln(&response)
		if response != "y" {
			return
		}
		params = append(params, "-n")
		err = new(gochips.PipedExec).
			Command(git, params...).
			Run(os.Stdout, os.Stdout)
	}
	gochips.ExitIfError(err)

	for i := 0; i < 2; i++ {
		_, sterr, err := new(gochips.PipedExec).
			Command(git, push).
			RunToStrings()
		if i == 0 && err != nil {
			if strings.Contains(sterr, "has no upstream") {
				remotelist := getRemotes()
				if len(remotelist) == 0 {
					fmt.Printf("\nRemote not found. It's somethig wrong with your repository\n ")
					return
				}
				brName := GetCurrentBranchName() // Suggest to execute git push --set-upstream origin <branch-name>
				var response string

				if len(remotelist) == 1 {
					fmt.Printf("\nCurrent branch has no upstream branch.\nI am going to execute 'git push --set-upstream origin %s'.\nAgree[y/n]? ", brName)
					fmt.Scanln(&response)
					if response == pushYes {
						setUpstreamBranch("origin", brName)
						continue
					}
				}
				fmt.Printf("\nCurrent branch has no upstream branch.")

				choice := ""
				prompt := &survey.Select{
					Message: "Choose a remote for your branch:",
					Options: remotelist,
				}
				err := survey.AskOne(prompt, &choice)
				if err != nil {
					fmt.Println(err.Error())
				}

				fmt.Printf("\nYour choice is %s.\nI am going to execute 'git push --set-upstream %s %s'.\nAgree[y/n]? ", choice, choice, brName)
				fmt.Scanln(&response)
				if response == pushYes {
					setUpstreamBranch(choice, brName)
					continue
				}
			}
		}
		break
	}

	gochips.ExitIfError(err)
}

// Download sources from git repo
func Download(cfg vcs.CfgDownload) {
	err := new(gochips.PipedExec).
		Command(git, "pull").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
}

// Gui shows gui
func Gui() {
	err := new(gochips.PipedExec).
		Command(git, "gui").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
}

func getFullRepoAndOrgName() string {
	stdouts, _, err := new(gochips.PipedExec).
		Command(git, "config", "--local", "remote.origin.url").
		Command("sed", "s/\\.git$//").
		RunToStrings()
	if err != nil {
		gochips.Error("getFullRepoAndOrgName error:", err)
		return ""
	}
	return strings.TrimSuffix(strings.TrimSpace(stdouts), "/")
}

// GetRepoAndOrgName - from .git/config
func GetRepoAndOrgName() (repo string, org string) {

	repourl := getFullRepoAndOrgName()
	arr := strings.Split(repourl, slash)
	if len(arr) > 0 {
		repo = arr[len(arr)-1]
	}

	if len(arr) > 1 {
		org = arr[len(arr)-2]
	}
	gochips.Verbose("GetRepoAndOrgName ok", "repourl:", repourl, "arr:", arr, "repo:", repo, "org:", org)
	return
}

func IsMainOrg() bool {
	_, org := GetRepoAndOrgName()
	return (org != getUserName())
}

// Fork repo
func Fork() (repo string, err error) {
	repo, org := GetRepoAndOrgName()
	if len(repo) == 0 {
		fmt.Println(repoNotFound)
		os.Exit(1)
	}
	remoteurl := GetRemoteUpstreamURL()
	if len(remoteurl) > 0 {
		return repo, errors.New(errAlreadyForkedMsg)
	}

	if !IsMainOrg() {
		return repo, errors.New(errAlreadyForkedMsg)
	}
	_, chExist := ChangedFilesExist()
	if chExist {
		err = new(gochips.PipedExec).
			Command(git, "add", ".").
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
		err = new(gochips.PipedExec).
			Command(git, "stash").
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
	}

	err = new(gochips.PipedExec).
		Command("gh", "repo", "fork", org+slash+repo, "--clone=false").
		Run(os.Stdout, os.Stdout)
	if err != nil {
		gochips.Error("Fork error:", err)
		return repo, err
	}
	return repo, nil
}

func GetRemoteUpstreamURL() string {
	stdouts, _, err := new(gochips.PipedExec).
		Command(git, "config", "--local", "remote.upstream.url").
		RunToStrings()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdouts)
}

func PopStashedFiles() {
	if !stashEntiresExist() {
		return
	}
	_, stderr, err := new(gochips.PipedExec).
		Command(git, "stash", "pop").
		RunToStrings()
	if err != nil {
		gochips.Error("PopStashedFiles error:", stderr)
	}
}

func GetMainBranch() string {
	_, _, err := new(gochips.PipedExec).
		Command(git, branch, "-r").
		Command("grep", "/main").
		RunToStrings()
	if err == nil {
		return "main"
	}
	return "master"
}

func getUserName() string {
	stdouts, _, err := new(gochips.PipedExec).
		Command(git, "config", "--global", "user.name").
		RunToStrings()
	gochips.ExitIfError(err)
	return strings.TrimSpace(stdouts)
}

func MakeUpstreamForBranch(parentrepo string) {
	_, _, err := new(gochips.PipedExec).
		Command(git, "remote", "add", "upstream", "https://github.com/"+parentrepo).
		RunToStrings()
	gochips.ExitIfError(err)
}

// MakeUpstream s.e.
func MakeUpstream(repo string) {
	user := getUserName()

	if len(user) == 0 {
		fmt.Println(userNotFound)
		os.Exit(1)
	}

	mainbranch := GetMainBranch()
	err := new(gochips.PipedExec).
		Command(git, "remote", "rename", "origin", "upstream").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
	err = new(gochips.PipedExec).
		Command(git, "remote", "add", "origin", "https://github.com/"+user+slash+repo).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
	time.Sleep(1 * time.Second)
	err = new(gochips.PipedExec).
		Command(git, "fetch", "origin").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
	err = new(gochips.PipedExec).
		Command(git, branch, "--set-upstream-to", "origin/"+mainbranch, mainbranch).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
}

// DevShort  - dev branch in trunk
func DevShort(branch string, comments []string) {
	mainbrach := GetMainBranch()
	_, chExist := ChangedFilesExist()
	var err error
	if chExist {
		err = new(gochips.PipedExec).
			Command(git, "add", ".").
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
		err = new(gochips.PipedExec).
			Command(git, "stash").
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
	}

	err = new(gochips.PipedExec).
		Command(git, "config", "pull.rebase", "false").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	err = new(gochips.PipedExec).
		Command(git, "pull", "upstream", mainbrach, "--no-edit").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	_, stderr, err := new(gochips.PipedExec).
		Command(git, "checkout", mainbrach).
		RunToStrings()
	if err != nil {
		if strings.Contains(err.Error(), "128") && strings.Contains(stderr, "matched multiple") {
			err = new(gochips.PipedExec).
				Command(git, "checkout", "--track", "origin/"+mainbrach).
				Run(os.Stdout, os.Stdout)
			gochips.ExitIfError(err)
		}
	}

	err = new(gochips.PipedExec).
		Command(git, "checkout", "-b", branch).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	// Add empty commit to create commit object and link notes to it
	err = new(gochips.PipedExec).
		Command(git, "commit", "--allow-empty", "-m", "Commit for keeping notes in branch").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
	addNotes(comments)

	err = new(gochips.PipedExec).
		Command(git, push, "-u", origin, branch).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	if chExist {
		err = new(gochips.PipedExec).
			Command(git, "stash", "pop").
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
	}
}

func GetIssuerepoFromUrl(url string) (reponame string) {
	if len(url) < 2 {
		return
	}
	if strings.HasSuffix(url, "/") {
		url = url[:len(url)-1]
	}

	arr := strings.Split(url, slash)
	if len(arr) > 5 {
		repo := arr[len(arr)-3]
		org := arr[len(arr)-4]
		reponame = org + "/" + repo
	}

	return
}

// Dev issue branch
func DevIssue(issueNumber int, args ...string) (branch string, notes []string) {
	repo, org := GetRepoAndOrgName()
	if len(repo) == 0 {
		fmt.Println(repoNotFound)
		os.Exit(1)
	}

	strissuenum := strconv.Itoa(issueNumber)
	myrepo := org + "/" + repo
	parentrepo := GetParentRepoName()
	if len(args) > 0 {
		url := args[0]
		issuerepo := GetIssuerepoFromUrl(url)
		if len(issuerepo) > 0 {
			parentrepo = issuerepo
		}
	}

	err := new(gochips.PipedExec).
		Command("gh", "repo", "set-default", myrepo).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	stdouts, stderr, err := new(gochips.PipedExec).
		Command("gh", "issue", "develop", strissuenum, "--issue-repo="+parentrepo, "--repo", myrepo).
		RunToStrings()
	if err != nil {
		fmt.Println(stderr)
		os.Exit(1)
		return
	}

	branch = strings.TrimSpace(stdouts)
	segments := strings.Split(branch, "/")
	branch = segments[len(segments)-1]

	if len(branch) == 0 {
		fmt.Println("Can not create branch for issue")
		os.Exit(1)
		return
	}

	issuename := GetIssueNameByNumber(strissuenum, parentrepo)
	comment := IssuePRTtilePrefix + " " + issuename
	body := ""
	if len(issuename) > 0 {
		body = "Resolves #" + strissuenum + " " + issuename
	}
	return branch, []string{comment, body}
}

func GetIssueNameByNumber(issueNum string, parentrepo string) string {
	stdouts, _, err := new(gochips.PipedExec).
		Command("gh", "issue", "view", issueNum, "--repo", parentrepo).
		Command("grep", "title:").
		Command("awk", "{ $1=\"\"; print substr($0, 2) }").
		RunToStrings()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdouts)
}

// Dev branch
func Dev(branch string, comments []string) {
	mainbrach := GetMainBranch()
	_, chExist := ChangedFilesExist()
	var err error
	if chExist {
		err = new(gochips.PipedExec).
			Command(git, "add", ".").
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
		err = new(gochips.PipedExec).
			Command(git, "stash").
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
	}

	err = new(gochips.PipedExec).
		Command(git, "config", "pull.rebase", "false").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	err = new(gochips.PipedExec).
		Command(git, "pull", "upstream", mainbrach, "--no-edit").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	_, stderr, err := new(gochips.PipedExec).
		Command(git, "checkout", mainbrach).
		RunToStrings()
	if err != nil {
		if strings.Contains(err.Error(), "128") && strings.Contains(stderr, "matched multiple") {
			err = new(gochips.PipedExec).
				Command(git, "checkout", "--track", "origin/"+mainbrach).
				Run(os.Stdout, os.Stdout)
			gochips.ExitIfError(err)
		}
	}
	gochips.ExitIfError(err)

	err = new(gochips.PipedExec).
		Command(git, "pull", "-p", "upstream", mainbrach).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
	err = new(gochips.PipedExec).
		Command(git, push).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	err = new(gochips.PipedExec).
		Command(git, "checkout", "-b", branch).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	// Add empty commit to create commit object and link notes to it
	err = new(gochips.PipedExec).
		Command(git, "commit", "--allow-empty", "-m", "Commit for keeping notes in branch").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	addNotes(comments)

	err = new(gochips.PipedExec).
		Command(git, push, "-u", origin, branch).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	if chExist {
		err = new(gochips.PipedExec).
			Command(git, "stash", "pop").
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
	}
}

func addNotes(comments []string) {
	if len(comments) == 0 {
		return
	}
	// Add new Notes
	for _, s := range comments {
		str := strings.TrimSpace(s)
		if len(str) > 0 {
			err := new(gochips.PipedExec).
				Command(git, "notes", "append", "-m", s).
				Run(os.Stdout, os.Stdout)
			gochips.ExitIfError(err)
		}
	}
}

func GetNotes() (notes []string, result bool) {
	stdouts, _, err := new(gochips.PipedExec).
		Command(git, "log", "--pretty=format:%N", "HEAD", "^main").
		Command("gawk", "{print $1}").
		Command("grep", "-v", "^$").
		Command("head", "-1").
		RunToStrings()
	if err != nil {
		return notes, false
	}
	notes = strings.Split(stdouts, "\n")
	return notes, true
}

// GetParentRepoName - parent repo of forked
func GetParentRepoName() (name string) {
	repo, org := GetRepoAndOrgName()
	stdouts, strerr, err := new(gochips.PipedExec).
		Command("gh", "api", "repos/"+org+"/"+repo, "--jq", ".parent.full_name").
		RunToStrings()
	if err != nil {
		fmt.Println("--------------------------------------------------------------------")
		fmt.Println(strerr)
		os.Exit(1)
	}
	name = strings.TrimSpace(stdouts)
	return
}

// IsBranchInMain Is my branch in main org?
func IsBranchInMain() bool {
	repo, org := GetRepoAndOrgName()
	parent := GetParentRepoName()
	return (parent == org+"/"+repo) || (strings.TrimSpace(parent) == "")
}

// GetMergedBranchList returns merged user's branch list
func GetMergedBranchList() (brlist []string, err error) {

	mbrlist := []string{}
	_, org := GetRepoAndOrgName()
	repo := GetParentRepoName()

	stdouts, _, err := new(gochips.PipedExec).
		Command("gh", "pr", "list", "-L", "200", "--state", "merged", "--author", org, "--repo", repo).
		RunToStrings()
	if err != nil {
		return []string{}, err
	}

	mbrlistraw := strings.Split(stdouts, "\n")
	for _, mbranchstr := range mbrlistraw {
		arr := strings.Split(mbranchstr, ":")
		if len(arr) > 1 {
			if strings.Contains(arr[1], "MERGED") {
				arrstr := strings.ReplaceAll(strings.TrimSpace(arr[1]), "MERGED", "")
				arrstr = strings.TrimSpace(arrstr)
				if !strings.Contains(arrstr, "master") && !strings.Contains(arrstr, "main") {
					mbrlist = append(mbrlist, arrstr)
				}
			}
		}
	}
	_, _, err = new(gochips.PipedExec).
		Command(git, "remote", "prune", origin).
		RunToStrings()
	gochips.ExitIfError(err)

	stdouts, _, err = new(gochips.PipedExec).
		Command(git, branch, "-r").
		RunToStrings()
	gochips.ExitIfError(err)
	mybrlist := strings.Split(stdouts, "\n")

	for _, mybranch := range mybrlist {
		mybranch := strings.TrimSpace(string(mybranch))
		mybranch = strings.ReplaceAll(strings.TrimSpace(mybranch), "origin/", "")
		mybranch = strings.TrimSpace(mybranch)
		bfound := false
		if strings.Contains(mybranch, "master") || strings.Contains(mybranch, "main") || strings.Contains(mybranch, "HEAD") {
			bfound = false
		} else {
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
func DeleteBranchesRemote(brs []string) {
	if len(brs) == 0 {
		return
	}
	for _, br := range brs {
		_, _, err := new(gochips.PipedExec).
			Command(git, push, origin, ":"+br).
			RunToStrings()
		if err != nil {
			fmt.Printf("Branch %s was not deleted\n", br)
		}
		gochips.ExitIfError(err)
		fmt.Printf("Branch %s deleted\n", br)
	}
}

// GetGoneBranchesLocal returns gone local branches
func GetGoneBranchesLocal() *[]string {
	// https://dev.heeus.io/launchpad/#!14544
	// 1. Step
	_, _, err := new(gochips.PipedExec).
		Command(git, fetch, "-p", "--dry-run").
		RunToStrings()
	gochips.ExitIfError(err)
	_, _, err = new(gochips.PipedExec).
		Command(git, fetch, "-p").
		RunToStrings()
	gochips.ExitIfError(err)
	// 2. Step
	mainbranch := GetMainBranch()
	_, _, err = new(gochips.PipedExec).
		Command(git, checkout, mainbranch).
		RunToStrings()
	gochips.ExitIfError(err)
	// 3. Step
	stdouts, _, err := new(gochips.PipedExec).
		Command(git, branch, "-vv").
		Command("grep", ": ").
		Command("gawk", "{print $1}").
		RunToStrings()
	if nil != err {
		return &[]string{}
	}
	strs := strings.Split(stdouts, "\n")
	return &strs
}

// DeleteBranchesLocal s.e.
func DeleteBranchesLocal(strs *[]string) {
	for _, str := range *strs {
		if strings.TrimSpace(str) != "" {
			_, _, err := new(gochips.PipedExec).
				Command(git, branch, "-D", str).
				RunToStrings()
			fmt.Printf("Branch %s deleted\n", str)
			gochips.ExitIfError(err)
		}
	}
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
					note = note + " " + s
				}
				if strings.Contains(s, IssuePRTtilePrefix) {
					break
				}
			}
		}
	}
	return note, url
}

func getBodyFromNotes(notes []string) string {
	b := ""
	if (len(notes) > 1) && strings.Contains(notes[0], IssuePRTtilePrefix) {
		for i, note := range notes {
			note = strings.TrimSpace(note)
			strings.Split(strings.ReplaceAll(note, "\r\n", "\n"), "")
			if i > 0 && len(note) > 0 {
				b = b + note
			}
		}
	}
	return b
}

// MakePR s.e.
func MakePR(notes []string, asDraft bool) (err error) {
	if len(notes) == 0 {
		return errors.New(errMsgPRNotesImpossible)
	}
	var strnotes string
	var url string
	strnotes, url = GetNoteAndURL(notes)
	b := getBodyFromNotes(notes)
	if len(b) == 0 {
		b = strnotes
	}
	if len(url) > 0 {
		b = b + "\n" + url
	}
	strbody := fmt.Sprintln(b)
	parentrepo := GetParentRepoName()
	if asDraft {
		err = new(gochips.PipedExec).
			Command("gh", "pr", "create", "--draft", "-t", strnotes, "-b", strbody, "-R", parentrepo).
			Run(os.Stdout, os.Stdout)
	} else {
		err = new(gochips.PipedExec).
			Command("gh", "pr", "create", "-t", strnotes, "-b", strbody, "-R", parentrepo).
			Run(os.Stdout, os.Stdout)
	}
	return err
}

// MakePRMerge merges Pull Request by URL
func MakePRMerge(prurl string) (err error) {
	if len(prurl) == 0 {
		return errors.New(errMsgPRMerge)
	}
	if !strings.Contains(prurl, "https") {
		return errors.New(errMsgPRBadFormat)
	}

	parentrepo := retrieveRepoNameFromUPL(prurl)
	var val *gchResponse
	// The checks could not found yet, need to wait for 1..10 seconds
	for idx := 0; idx < 5; idx++ {
		val = waitPRChecks(parentrepo, prurl)
		if prCheckAbsent(val) {
			time.Sleep(2 * time.Second)
			fmt.Println(msgPRCheckNotFoundYet)
			continue
		}
		break
	}

	// If after 10 seconds  we still have no checks - then thery don't exists at all, so we can merge
	if prCheckAbsent(val) {
		fmt.Println(" ")
		fmt.Println(msgPRCheckNotFound)
		fmt.Println(" ")
	} else {
		if len(val._stderr) > 0 {
			return errors.New(val._stderr)
		}
		if val._err != nil {
			return val._err
		}
		if !prCheckSuccess(val) {
			return errors.New(errUnknowGHResponse + ": " + val._stdout)
		}
	}

	if len(parentrepo) == 0 {
		err = new(gochips.PipedExec).
			Command("gh", "pr", "merge", prurl, "--squash").
			Run(os.Stdout, os.Stdout)
	} else {
		err = new(gochips.PipedExec).
			Command("gh", "pr", "merge", prurl, "--squash", "-R", parentrepo).
			Run(os.Stdout, os.Stdout)
		if err != nil {
			return err
		}

		repo, org := GetRepoAndOrgName()
		if len(repo) > 0 {
			err = new(gochips.PipedExec).
				Command("gh", "repo", "sync", org+"/"+repo).
				Run(os.Stdout, os.Stdout)
		}

	}
	return err
}

func retrieveRepoNameFromUPL(prurl string) string {
	var strs []string = strings.Split(prurl, "/")
	if len(strs) < 4 {
		return ""
	}
	res := ""
	lenstr := len(strs)
	for i := lenstr - 4; i < lenstr-2; i++ {
		if res == "" {
			res = strs[i]
		} else {
			res = res + "/" + strs[i]
		}
	}
	return res
}

func prCheckAbsent(val *gchResponse) bool {
	return strings.Contains(val._stderr, nochecksmsg)
}

func prCheckSuccess(val *gchResponse) bool {
	ss := strings.Split(val._stdout, "\n")
	for _, s := range ss {
		if strings.Contains(s, "build") && strings.Contains(s, "pass") {
			return true
		}
	}
	return false
}

func waitPRChecks(parentrepo string, prurl string) *gchResponse {
	c := make(chan *gchResponse)

	// Run checking status of PR Checks
	go runPRChecksChecks(parentrepo, prurl, c)

	strw := msgWaitingPR
	var val *gchResponse
	var ok bool
	waitTimer := time.NewTimer(40 * time.Second)
	fmt.Print(strw)
	for {
		select {
		case val, ok = <-c:
			fmt.Println("")
			if ok {
				return val
			}
			return &gchResponse{_err: errors.New(errSomethigWrong)}
		case <-waitTimer.C:
			fmt.Println("")
			return &gchResponse{_err: errors.New(errTimer40Sec)}
		default:
			time.Sleep(time.Second)
			fmt.Print(".")
		}
	}
}

func runPRChecksChecks(parentrepo string, prurl string, c chan *gchResponse) {
	var stdout, stderr string
	var err error
	if len(parentrepo) == 0 {
		stdout, stderr, err = new(gochips.PipedExec).
			Command("gh", "pr", "checks", prurl, "--watch").
			RunToStrings()
	} else {
		stdout, stderr, err = new(gochips.PipedExec).
			Command("gh", "pr", "checks", prurl, "--watch", "-R", parentrepo).
			RunToStrings()
	}
	c <- &gchResponse{stdout, stderr, err}
}

func GetCurrentBranchName() string {
	stdout, _, _ := new(gochips.PipedExec).
		Command(git, branch).
		Command("sed", "-n", "/\\* /s///p").
		RunToStrings()
	return strings.TrimSpace(stdout)
}

// getRemotes shows list of names of all remotes
func getRemotes() []string {
	stdout, _, _ := new(gochips.PipedExec).
		Command(git, "remote").
		RunToStrings()
	strs := strings.Split(stdout, "\n")
	for i, str := range strs {
		if len(strings.TrimSpace(str)) == 0 {
			strs = append(strs[:i], strs[i+1:]...)
		}
	}
	return strs
}

// GetFilesForCommit shows list of file names, ready for commit
func GetFilesForCommit() []string {
	stdout, _, _ := new(gochips.PipedExec).
		Command(git, "status", "-s").
		RunToStrings()
	ss := strings.Split(stdout, "\n")
	var strs []string
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			strs = append(strs, s)
		}
	}
	return strs
}

func setUpstreamBranch(repo string, branch string) {
	if branch == "" {
		branch = "main"
	}
	errupstream := new(gochips.PipedExec).
		Command(git, "push", "--set-upstream", repo, branch).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(errupstream)
}

// GetCommitFileSizes returns quantity of cmmited files and their total sizes
func GetCommitFileSizes() (totalSize int, quantity int) {
	totalSize = 0
	quantity = 0
	stdout, _, err := new(gochips.PipedExec).
		Command(git, "status", "--porcelain").
		Command("awk", "{if ($1 == \"??\") print $2}").
		RunToStrings()
	gochips.ExitIfError(err)
	files := strings.Split(stdout, "\n")

	if len(files) == 0 {
		return
	}

	for _, file := range files {
		if len(file) > 0 {

			stdout, _, err = new(gochips.PipedExec).
				Command("wc", "-c", file).
				Command("awk", "{print $1}").
				RunToStrings()
			gochips.ExitIfError(err)

			strval := strings.TrimSpace(stdout)
			if strval != "" {
				sz, err := strconv.Atoi(strval)
				if err != nil {
					fmt.Println("Error during conversion of value: ", err.Error())
					return
				}
				totalSize = totalSize + sz
				quantity = quantity + 1
			}
		}
	}
	return totalSize, quantity
}

func getGlobalHookFolder() string {
	stdout, _, err := new(gochips.PipedExec).
		Command(git, "config", "--global", "core.hooksPath").
		RunToStrings()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdout)
}

func getLocalHookFolder() string {
	dir := GetRootFolder()
	filename := "/.git/hooks/pre-commit"
	filepath := dir + filename
	return strings.TrimSpace(filepath)
}

// GlobalPreCommitHookExist - s.e.
func GlobalPreCommitHookExist() bool {
	filepath := getGlobalHookFolder()
	if len(filepath) == 0 {
		return false // global hook folder not defined
	}
	err := os.MkdirAll(filepath, os.ModePerm)
	gochips.ExitIfError(err)

	filepath = filepath + "/pre-commit"
	// Check if the file already exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return false // File pre-commit does not exist
	}
	return largFileHookExist(filepath)
}

// LocalPreCommitHookExist - s.e.
func LocalPreCommitHookExist() bool {
	filepath := getLocalHookFolder()
	// Check if the file already exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return false
	}
	return largFileHookExist(filepath)
}

func largFileHookExist(filepath string) bool {
	substring := "large-file-hook.sh"

	_, _, err := new(gochips.PipedExec).
		Command("grep", "-l", substring, filepath).
		RunToStrings()
	return err == nil
}

// SetGlobalPreCommitHook - s.e.
func SetGlobalPreCommitHook() {
	var err error
	path := getGlobalHookFolder()

	if len(path) == 0 {
		rootUser, err := user.Current()
		gochips.ExitIfError(err)
		path = rootUser.HomeDir
		path = path + "/.git/hooks"
		err = os.MkdirAll(path, os.ModePerm)
		gochips.ExitIfError(err)
	}

	// Set global hooks folder
	err = new(gochips.PipedExec).
		Command(git, "config", "--global", "core.hookspath", path).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	filepath := path + "/pre-commit"
	f := createOrOpenFile(filepath)
	f.Close()
	if !largFileHookExist(filepath) {
		fillPreCommitFile(filepath)
	}
}

func GetRootFolder() string {
	stdouts, _, err := new(gochips.PipedExec).
		Command(git, "rev-parse", "--show-toplevel").
		RunToStrings()
	gochips.ExitIfError(err)
	return strings.TrimSpace(stdouts)
}

// SetLocalPreCommitHook - s.e.
func SetLocalPreCommitHook() {

	// Turn off globa1 hooks
	new(gochips.PipedExec).
		Command(git, "config", "--global", "--unset", "core.hookspath").
		Run(os.Stdout, os.Stdout)

	dir := GetRootFolder()
	filename := "/.git/hooks/pre-commit"
	filepath := dir + filename

	// Check if the file already exists
	f := createOrOpenFile(filepath)
	f.Close()

	if !largFileHookExist(filepath) {
		fillPreCommitFile(filepath)
	}
}

func createOrOpenFile(filepath string) *os.File {
	_, err := os.Stat(filepath)
	var f *os.File
	if os.IsNotExist(err) {
		// Create file pre-commit
		f, err = os.Create(filepath)
		gochips.ExitIfError(err)
		_, err = f.WriteString("#!/bin/bash\n")
	} else {
		f, err = os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, 0644)
	}
	gochips.ExitIfError(err)
	return f
}

func fillPreCommitFile(myfilepath string) {
	f := createOrOpenFile(myfilepath)
	defer f.Close()

	pathLaregFile := "https://raw.githubusercontent.com/untillpro/ci-action/master/scripts/large-file-hook.sh"

	dir := GetRootFolder()
	fname := "/.git/hooks/large-file-hook.sh"
	lf := dir + fname

	err := new(gochips.PipedExec).
		Command("curl", "-s", "-o", lf, pathLaregFile).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	hookcode := "\n#Here is large files commit prevent is added by [qs]\n"
	hookcode = hookcode + "bash " + lf + "\n"
	_, err = f.WriteString(hookcode)
	gochips.ExitIfError(err)

	//cmd := exec.Command("chmod +x " + myfilepath)
	//err = cmd.Run()

	err = new(gochips.PipedExec).
		Command("chmod", "+x", myfilepath).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
}

func UpstreamNotExist(repo string) bool {
	remotelist := getRemotes()
	return len(remotelist) < 2
}

func PRAhead() bool {
	brName := GetCurrentBranchName()
	mainbr := GetMainBranch()

	remotelist := getRemotes()
	if len(remotelist) < 2 {
		gochips.ExitIfError(errors.New("Upstream is not set, Pull Request can not be done."))
	}

	err := new(gochips.PipedExec).
		Command(git, "fetch", "upstream").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	_, _, err = new(gochips.PipedExec).
		Command(git, "diff", "--quiet", brName+"...upstream/"+mainbr).
		RunToStrings()

	return err != nil

}

func MergeFromUpstream() {
	mainbr := GetMainBranch()

	err := new(gochips.PipedExec).
		Command(git, "config", "pull.rebase", "false").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	err = new(gochips.PipedExec).
		Command(git, "pull", "upstream", mainbr, "--no-edit").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	brName := GetCurrentBranchName()
	err = new(gochips.PipedExec).
		Command(git, "push", "origin", brName).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
}

// GHInstalled returns is gh utility installed
func GHInstalled() bool {
	_, _, err := new(gochips.PipedExec).
		Command("gh", "--version").
		RunToStrings()
	return err == nil
}

// GHLoggedIn returns is gh logged in
func GHLoggedIn() bool {
	_, _, err := new(gochips.PipedExec).
		Command("gh", "auth", "status").
		RunToStrings()
	return err == nil
}

func GetInstalledQSVersion() string {
	stdouts, stderr, err := new(gochips.PipedExec).
		Command("go", "env", "GOPATH").
		RunToStrings()
	if err != nil {
		gochips.Error("GetInstalledVersion error:", stderr)
	}

	gopath := strings.TrimSpace(stdouts)
	if len(gopath) == 0 {
		gochips.Error("GetInstalledVersion error:", errors.New("GOPATH is not defined"))
	}
	qsexe := "qs"
	if runtime.GOOS == "windows" {
		qsexe = "qs.exe"
	}

	stdouts, stderr, err = new(gochips.PipedExec).
		Command("go", "version", "-m", gopath+"/bin/"+qsexe).
		Command("grep", "-i", "-h", "mod.*github.com/untillpro/qs").
		Command("awk", "{print $3}").
		RunToStrings()
	if err != nil {
		gochips.Error("GetInstalledQSVersion error:", stderr)
	}
	return strings.TrimSpace(stdouts)
}

func GetLastQSVersion() string {
	stdouts, stderr, err := new(gochips.PipedExec).
		Command("go", "list", "-m", "-versions", "github.com/untillpro/qs").
		RunToStrings()
	if err != nil {
		gochips.Error("GetLastQSVersion error:", stderr)
	}

	arr := strings.Split(strings.TrimSpace(stdouts), " ")
	if len(arr) == 0 {
		return ""
	}

	return arr[len(arr)-1]
}

func GetIssueNumFromBranchName(parentrepo string) (issuenum string, ok bool) {
	stdouts, stderr, err := new(gochips.PipedExec).
		Command("gh", "issue", "list", "-R", parentrepo).
		Command("gawk", "{print $1}").
		RunToStrings()
	if err != nil {
		gochips.Error("GetIssueNumFromBranchName:", stderr)
		return "", false
	}
	issuenums := strings.Split(stdouts, "\n")
	myfullbranchname := strings.TrimSpace(getFullRepoAndOrgName() + "/tree/" + GetCurrentBranchName())
	fmt.Println("Searching linked issue ")
	for _, issuenum := range issuenums {
		if len(issuenum) > 0 {
			fmt.Println("  Issue number: ", issuenum, "...")
			stdouts1, _, err1 := new(gochips.PipedExec).
				Command("gh", "issue", "develop", issuenum, "--list", "-R", parentrepo).
				Command("gawk", "{print $2}").
				RunToStrings()
			if (err1 == nil) && (len(stdouts1) > 10) {
				linkedbranches := strings.Split(stdouts1, "\n")
				for _, linkedbranche := range linkedbranches {
					if strings.EqualFold(myfullbranchname, linkedbranche) {
						return issuenum, true
					}
				}
			}
		}
	}
	return "", false
}

func GetIssuePRTitle(issueNum string, parentrepo string) []string {
	name := GetIssueNameByNumber(issueNum, parentrepo)
	s := IssuePRTtilePrefix + " " + name
	body := "Resolves #" + issueNum + " " + name
	return []string{s, body}
}
