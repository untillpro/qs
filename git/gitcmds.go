package git

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/utils"
	"github.com/untillpro/qs/vcs"
)

const (
	mimm                  = "-m"
	slash                 = "/"
	caret                 = "\n"
	git                   = "git"
	push                  = "push"
	pull                  = "pull"
	fetch                 = "fetch"
	branch                = "branch"
	checkout              = "checkout"
	origin                = "origin"
	originSlash           = "origin/"
	httppref              = "https"
	pushYes               = "y"
	nochecksmsg           = "no checks reported"
	msgWaitingPR          = "Waiting PR checks.."
	msgPRCheckNotFoundYet = "..not found yet"
	msgPRCheckNotFound    = "No checks for PR found, merge without checks"
	MsgPreCommitError     = "Attempt to commit too"
	MsgCommitForNotes     = "Commit for keeping notes in branch"
	oneSpace              = " "
	err128                = "128"

	repoNotFound            = "git repo name not found"
	userNotFound            = "git user name not found"
	ErrAlreadyForkedMsg     = "you are in fork already\nExecute 'qs dev [branch name]' to create dev branch"
	ErrMsgPRNotesImpossible = "pull request without comments is impossible"
	ErrMsgPRMerge           = "URL of PR is needed"
	ErrMsgPRBadFormat       = "pull request URL has bad format"
	ErrTimer40Sec           = "time out 40 seconds"
	ErrSomethigWrong        = "something went wrong"
	ErrUnknowGHResponse     = "unknown response from gh"
	PushDefaultMsg          = "misc"
	mainBrachName           = "main"

	IssuePRTtilePrefix = "Resolves issue"
	IssueSign          = "Resolves #"

	prTimeWait                     = 40
	minIssueNoteLength             = 10
	minRepoNameLength              = 4
	bashFilePerm       os.FileMode = 0644
	timeWaitPR                     = 5

	issuelineLength  = 5
	issuelinePosOrg  = 4
	issuelinePosRepo = 3
)

type gchResponse struct {
	_stdout string
	_stderr string
	_err    error
}

// ExitIfFalse s.e.
func ExitIfFalse(cond bool, args ...interface{}) {
	if !cond {
		fmt.Fprintln(os.Stderr, args...)
		os.Exit(1)
	}
}

// ExitIfError s.e.
func ExitIfError(err error, args ...interface{}) {
	if nil != err {
		fmt.Fprintln(os.Stderr, args...)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func CheckIfGitRepo() string {
	stdouts, _, err := new(exec.PipedExec).
		Command("git", "status", "-s").
		RunToStrings()
	if err != nil {
		if strings.Contains(err.Error(), err128) {
			err = errors.New("this is not a git repository")
		}
	}
	ExitIfError(err)
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
	stdouts, _, err := new(exec.PipedExec).
		Command(git, "stash", "list").
		RunToStrings()
	ExitIfError(err)
	stashentires := strings.TrimSpace(stdouts)
	return len(stashentires) > 0
}

// Status shows git repo status
func Status(cfg vcs.CfgStatus) {
	stdout, _, err := new(exec.PipedExec).
		Command("git", "remote", "-v").
		Command("grep", fetch).
		Command("sed", "s/(fetch)//").
		RunToStrings()
	if err != nil {
		if strings.Contains(err.Error(), err128) {
			err = errors.New("this is not a git repository")
		}
	}
	ExitIfError(err)
	fmt.Print(stdout)
	err = new(exec.PipedExec).
		Command("git", "status", "-s", "-b", "-uall").
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)
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
	logger.Info("Pulling")
	err := new(exec.PipedExec).
		Command("git", pull).
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)

	// *************************************************
	logger.Info("Reading current version")
	currentVersion, err := utils.ReadVersion()
	ExitIfError(err, "Error reading file 'version'")
	ExitIfFalse(len(currentVersion.PreRelease) > 0, "pre-release part of version does not exist: "+currentVersion.String())

	// Calculate target version

	targetVersion := currentVersion
	targetVersion.PreRelease = ""

	fmt.Printf("Version %v will be tagged, bumped and pushed, agree? [y]", targetVersion)
	var response string
	fmt.Scanln(&response)
	ExitIfFalse(response == "y")

	// *************************************************
	logger.Info("Updating 'version' file")
	ExitIfError(targetVersion.Save())

	// *************************************************
	logger.Info("Committing target version")
	{
		params := []string{"commit", "-a", mimm, "#scm-ver " + targetVersion.String()}
		err = new(exec.PipedExec).
			Command(git, params...).
			Run(os.Stdout, os.Stdout)
		ExitIfError(err)
	}

	// *************************************************
	logger.Info("Tagging")
	{
		tagName := "v" + targetVersion.String()
		n := time.Now()
		params := []string{"tag", mimm, "Version " + tagName + " of " + n.Format("2006/01/02 15:04:05"), tagName}
		err = new(exec.PipedExec).
			Command(git, params...).
			Run(os.Stdout, os.Stdout)
		ExitIfError(err)
	}

	// *************************************************
	logger.Info("Bumping version")
	newVersion := currentVersion
	{
		newVersion.Minor++
		newVersion.PreRelease = "SNAPSHOT"
		ExitIfError(newVersion.Save())
	}

	// *************************************************
	logger.Info("Committing new version")
	{
		params := []string{"commit", "-a", mimm, "#scm-ver " + newVersion.String()}
		err = new(exec.PipedExec).
			Command(git, params...).
			Run(os.Stdout, os.Stdout)
		ExitIfError(err)
	}

	// *************************************************
	logger.Info("Pushing to origin")
	{
		params := []string{push, "--follow-tags", origin}
		err = new(exec.PipedExec).
			Command(git, params...).
			Run(os.Stdout, os.Stdout)
		ExitIfError(err)
	}
}

// Upload upload sources to git repo
func Upload(cfg vcs.CfgUpload) {
	err := new(exec.PipedExec).
		Command(git, "add", ".").
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)

	params := []string{"commit", "-a"}
	for _, m := range cfg.Message {
		params = append(params, mimm, m)
	}
	_, sterr, err := new(exec.PipedExec).
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
		err = new(exec.PipedExec).
			Command(git, params...).
			Run(os.Stdout, os.Stdout)
	}
	ExitIfError(err)

	for i := 0; i < 2; i++ {
		_, sterr, err := new(exec.PipedExec).
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

	ExitIfError(err)
}

// Download sources from git repo
func Download(cfg vcs.CfgDownload) {
	err := new(exec.PipedExec).
		Command(git, pull).
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)
}

// Gui shows gui
func Gui() {
	err := new(exec.PipedExec).
		Command(git, "gui").
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)
}

func getFullRepoAndOrgName() string {
	stdouts, _, err := new(exec.PipedExec).
		Command(git, "config", "--local", "remote.origin.url").
		Command("sed", "s/\\.git$//").
		RunToStrings()
	if err != nil {
		logger.Error("getFullRepoAndOrgName error:", err)
		return ""
	}
	return strings.TrimSuffix(strings.TrimSpace(stdouts), slash)
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
		return repo, errors.New(ErrAlreadyForkedMsg)
	}

	if !IsMainOrg() {
		return repo, errors.New(ErrAlreadyForkedMsg)
	}
	_, chExist := ChangedFilesExist()
	if chExist {
		err = new(exec.PipedExec).
			Command(git, "add", ".").
			Run(os.Stdout, os.Stdout)
		ExitIfError(err)
		err = new(exec.PipedExec).
			Command(git, "stash").
			Run(os.Stdout, os.Stdout)
		ExitIfError(err)
	}

	err = new(exec.PipedExec).
		Command("gh", "repo", "fork", org+slash+repo, "--clone=false").
		Run(os.Stdout, os.Stdout)
	if err != nil {
		logger.Error("Fork error:", err)
		return repo, err
	}
	return repo, nil
}

// GetUserEmail - github user email
func GetUserEmail() string {
	stdouts, strerr, err := new(exec.PipedExec).
		Command("gh", "api", "user", "--jq", ".email").
		RunToStrings()
	if err != nil {
		fmt.Println("--------------------------------------------------------------------")
		fmt.Println(strerr)
		os.Exit(1)
	}
	return strings.TrimSpace(stdouts)
}

func GetRemoteUpstreamURL() string {
	stdouts, _, err := new(exec.PipedExec).
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
	_, stderr, err := new(exec.PipedExec).
		Command(git, "stash", "pop").
		RunToStrings()
	if err != nil {
		logger.Error("PopStashedFiles error:", stderr)
	}
}

func GetMainBranch() string {
	_, _, err := new(exec.PipedExec).
		Command(git, branch, "-r").
		Command("grep", "/main").
		RunToStrings()
	if err == nil {
		return mainBrachName
	}
	return "master"
}

func getUserName() string {
	stdouts, _, err := new(exec.PipedExec).
		Command("gh", "api", "user").
		Command("jq", "-r", ".login").
		RunToStrings()
	ExitIfError(err)
	return strings.TrimSpace(stdouts)
}

func MakeUpstreamForBranch(parentrepo string) {
	_, _, err := new(exec.PipedExec).
		Command(git, "remote", "add", "upstream", "https://github.com/"+parentrepo).
		RunToStrings()
	ExitIfError(err)
}

// MakeUpstream s.e.
func MakeUpstream(repo string) {
	user := getUserName()

	if len(user) == 0 {
		fmt.Println(userNotFound)
		os.Exit(1)
	}

	mainbranch := GetMainBranch()
	err := new(exec.PipedExec).
		Command(git, "remote", "rename", "origin", "upstream").
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)
	err = new(exec.PipedExec).
		Command(git, "remote", "add", "origin", "https://github.com/"+user+slash+repo).
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)
	time.Sleep(1 * time.Second)
	err = new(exec.PipedExec).
		Command(git, "fetch", "origin").
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)
	err = new(exec.PipedExec).
		Command(git, branch, "--set-upstream-to", originSlash+mainbranch, mainbranch).
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)
}

func GetIssuerepoFromUrl(url string) (reponame string) {
	if len(url) < 2 {
		return
	}
	if strings.HasSuffix(url, slash) {
		url = url[:len(url)-1]
	}

	arr := strings.Split(url, slash)
	if len(arr) > issuelineLength {
		repo := arr[len(arr)-issuelinePosRepo]
		org := arr[len(arr)-issuelinePosOrg]
		reponame = org + slash + repo
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
	myrepo := org + slash + repo
	parentrepo := GetParentRepoName()
	if len(args) > 0 {
		url := args[0]
		issuerepo := GetIssuerepoFromUrl(url)
		if len(issuerepo) > 0 {
			parentrepo = issuerepo
		}
	}

	err := new(exec.PipedExec).
		Command("gh", "repo", "set-default", myrepo).
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)

	stdouts, stderr, err := new(exec.PipedExec).
		Command("gh", "issue", "develop", strissuenum, "--issue-repo="+parentrepo, "--repo", myrepo).
		RunToStrings()
	if err != nil {
		fmt.Println(stderr)
		os.Exit(1)
		return
	}

	branch = strings.TrimSpace(stdouts)
	segments := strings.Split(branch, slash)
	branch = segments[len(segments)-1]

	if len(branch) == 0 {
		fmt.Println("Can not create branch for issue")
		os.Exit(1)
		return
	}

	issuename := GetIssueNameByNumber(strissuenum, parentrepo)
	comment := IssuePRTtilePrefix + " '" + issuename + "' "
	body := ""
	if len(issuename) > 0 {
		body = IssueSign + strissuenum + oneSpace + issuename
	}
	return branch, []string{comment, body}
}

func GetIssueNameByNumber(issueNum string, parentrepo string) string {
	stdouts, _, err := new(exec.PipedExec).
		Command("gh", "issue", "view", issueNum, "--repo", parentrepo).
		Command("grep", "title:").
		Command("gawk", "{ $1=\"\"; print substr($0, 2) }").
		RunToStrings()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdouts)
}

// Dev branch
func Dev(branch string, comments []string, branchinfork bool) {
	mainbrach := GetMainBranch()
	_, chExist := ChangedFilesExist()
	var err error
	if chExist {
		err = new(exec.PipedExec).
			Command(git, "add", ".").
			Run(os.Stdout, os.Stdout)
		ExitIfError(err)
		err = new(exec.PipedExec).
			Command(git, "stash").
			Run(os.Stdout, os.Stdout)
		ExitIfError(err)
	}

	err = new(exec.PipedExec).
		Command(git, "config", "pull.rebase", "true").
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)

	err = new(exec.PipedExec).
		Command(git, pull, "upstream", mainbrach, "--no-edit").
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)

	_, stderr, err := new(exec.PipedExec).
		Command(git, "checkout", mainbrach).
		RunToStrings()

	if err != nil {
		if strings.Contains(err.Error(), err128) && strings.Contains(stderr, "matched multiple") {
			err = new(exec.PipedExec).
				Command(git, "checkout", "--track", originSlash+mainbrach).
				Run(os.Stdout, os.Stdout)
			ExitIfError(err)
		}
	}
	ExitIfError(err)

	if branchinfork {
		err = new(exec.PipedExec).
			Command(git, pull, "-p", "upstream", mainbrach).
			Run(os.Stdout, os.Stdout)
		ExitIfError(err)
		err = new(exec.PipedExec).
			Command(git, push, origin, mainbrach).
			Run(os.Stdout, os.Stdout)
		ExitIfError(err)
	}

	err = new(exec.PipedExec).
		Command(git, "checkout", "-B", branch).
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)

	// Add empty commit to create commit object and link notes to it
	err = new(exec.PipedExec).
		Command(git, "commit", "--allow-empty", "-m", MsgCommitForNotes).
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)

	// Add empty commit to create commit object and link notes to it
	addNotes(comments)

	err = new(exec.PipedExec).
		Command(git, push, "-u", origin, branch).
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)

	if chExist {
		err = new(exec.PipedExec).
			Command(git, "stash", "pop").
			Run(os.Stdout, os.Stdout)
		ExitIfError(err)
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
			err := new(exec.PipedExec).
				Command(git, "notes", "append", "-m", s).
				Run(os.Stdout, os.Stdout)
			ExitIfError(err)
		}
	}
}

func GetNotes() (notes []string, result bool) {
	stdouts, _, err := new(exec.PipedExec).
		Command(git, "log", "--pretty=format:%N", "HEAD", "^main").
		RunToStrings()
	if err != nil {
		return notes, false
	}
	rawnotes := strings.Split(stdouts, caret)
	for _, rawnote := range rawnotes {
		note := strings.TrimSpace(rawnote)
		if len(note) > 0 {
			notes = append(notes, note)
		}
	}
	if len(notes) == 0 {
		return notes, false
	}
	return notes, true
}

// GetParentRepoName - parent repo of forked
func GetParentRepoName() (name string) {
	repo, org := GetRepoAndOrgName()
	stdouts, strerr, err := new(exec.PipedExec).
		Command("gh", "api", "repos/"+org+slash+repo, "--jq", ".parent.full_name").
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
	return (parent == org+slash+repo) || (strings.TrimSpace(parent) == "")
}

// GetMergedBranchList returns merged user's branch list
func GetMergedBranchList() (brlist []string, err error) {

	mbrlist := []string{}
	_, org := GetRepoAndOrgName()
	repo := GetParentRepoName()

	stdouts, _, err := new(exec.PipedExec).
		Command("gh", "pr", "list", "-L", "200", "--state", "merged", "--author", org, "--repo", repo).
		Command("gawk", "/MERGED/{print $4}").
		Command("gawk", "-F:", "{print $2}").
		Command("grep", "-v", "'^$'").
		RunToStrings()
	if err != nil {
		return []string{}, err
	}

	mbrlistraw := strings.Split(stdouts, caret)
	for _, mbranchstr := range mbrlistraw {
		arrstr := strings.TrimSpace(mbranchstr)
		if (strings.TrimSpace(arrstr) != "") && !strings.Contains(arrstr, "master") && !strings.Contains(arrstr, mainBrachName) {
			mbrlist = append(mbrlist, arrstr)
		}
	}
	_, _, err = new(exec.PipedExec).
		Command(git, "remote", "prune", origin).
		RunToStrings()
	ExitIfError(err)

	stdouts, _, err = new(exec.PipedExec).
		Command(git, branch, "-r").
		RunToStrings()
	ExitIfError(err)
	mybrlist := strings.Split(stdouts, caret)

	for _, mybranch := range mybrlist {
		mybranch := strings.TrimSpace(mybranch)
		mybranch = strings.ReplaceAll(strings.TrimSpace(mybranch), originSlash, "")
		mybranch = strings.TrimSpace(mybranch)
		bfound := false
		if strings.Contains(mybranch, "master") || strings.Contains(mybranch, mainBrachName) || strings.Contains(mybranch, "HEAD") {
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
		_, _, err := new(exec.PipedExec).
			Command(git, push, origin, ":"+br).
			RunToStrings()
		if err != nil {
			fmt.Printf("Branch %s was not deleted\n", br)
		}
		ExitIfError(err)
		fmt.Printf("Branch %s deleted\n", br)
	}
}

func PullUpstream() {
	mainbr := GetMainBranch()
	err := new(exec.PipedExec).
		Command(git, pull, "upstream", mainbr, "--no-edit").
		Run(os.Stdout, os.Stdout)

	if err != nil {
		parentrepo := GetParentRepoName()
		MakeUpstreamForBranch(parentrepo)
	}
}

// GetGoneBranchesLocal returns gone local branches
func GetGoneBranchesLocal() *[]string {
	// https://dev.heeus.io/launchpad/#!14544
	// 1. Step
	_, _, err := new(exec.PipedExec).
		Command(git, fetch, "-p", "--dry-run").
		RunToStrings()
	ExitIfError(err)
	_, _, err = new(exec.PipedExec).
		Command(git, fetch, "-p").
		RunToStrings()
	ExitIfError(err)
	// 2. Step
	stdouts, _, err := new(exec.PipedExec).
		Command(git, branch, "-vv").
		Command("grep", ":").
		Command("sed", "-n", "2p").
		Command("gawk", "{print $1}").
		RunToStrings()
	if nil != err {
		return &[]string{}
	}
	strs := strings.Split(stdouts, caret)
	return &strs
}

// DeleteBranchesLocal s.e.
func DeleteBranchesLocal(strs *[]string) {
	for _, str := range *strs {
		if strings.TrimSpace(str) != "" {
			_, _, err := new(exec.PipedExec).
				Command(git, branch, "-D", str).
				RunToStrings()
			fmt.Printf("Branch %s deleted\n", str)
			ExitIfError(err)
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
					note = note + oneSpace + s
				}
				if strings.Contains(strings.ToLower(s), strings.ToLower(IssuePRTtilePrefix)) {
					break
				}
			}
		}
	}
	return note, url
}

func GetBodyFromNotes(notes []string) string {
	b := ""
	if (len(notes) > 1) && strings.Contains(strings.ToLower(notes[0]), strings.ToLower(IssuePRTtilePrefix)) {
		for i, note := range notes {
			note = strings.TrimSpace(note)
			if (strings.Contains(note, "https://") && !strings.Contains(note, "/issues/")) || !strings.Contains(note, "https://") {
				strings.Split(strings.ReplaceAll(note, "\r\n", caret), "")
				if i > 0 && len(note) > 0 {
					b += note
				}
			}
		}
	}
	return b
}

// MakePR s.e.
func MakePR(title string, notes []string, asDraft bool) (err error) {
	if len(notes) == 0 {
		return errors.New(ErrMsgPRNotesImpossible)
	}
	var strnotes string
	var url string
	strnotes, url = GetNoteAndURL(notes)
	b := GetBodyFromNotes(notes)
	if len(b) == 0 {
		b = strnotes
	}
	if len(url) > 0 {
		b = b + caret + url
	}

	strbody := fmt.Sprintln(b)
	parentrepo := GetParentRepoName()
	if asDraft {
		err = new(exec.PipedExec).
			Command("gh", "pr", "create", "--draft", "-t", title, "-b", strbody, "-R", parentrepo).
			Run(os.Stdout, os.Stdout)
	} else {
		err = new(exec.PipedExec).
			Command("gh", "pr", "create", "-t", title, "-b", strbody, "-R", parentrepo).
			Run(os.Stdout, os.Stdout)
	}
	return err
}

// MakePRMerge merges Pull Request by URL
func MakePRMerge(prurl string) (err error) {
	if len(prurl) == 0 {
		return errors.New(ErrMsgPRMerge)
	}
	if !strings.Contains(prurl, "https") {
		return errors.New(ErrMsgPRBadFormat)
	}

	parentrepo := retrieveRepoNameFromUPL(prurl)
	var val *gchResponse
	// The checks could not found yet, need to wait for 1..10 seconds
	for idx := 0; idx < timeWaitPR; idx++ {
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
		fmt.Println(oneSpace)
		fmt.Println(msgPRCheckNotFound)
		fmt.Println(oneSpace)
	} else {
		if len(val._stderr) > 0 {
			return errors.New(val._stderr)
		}
		if val._err != nil {
			return val._err
		}
		if !prCheckSuccess(val) {
			return errors.New(ErrUnknowGHResponse + ": " + val._stdout)
		}
	}

	if len(parentrepo) == 0 {
		err = new(exec.PipedExec).
			Command("gh", "pr", "merge", prurl, "--squash").
			Run(os.Stdout, os.Stdout)
	} else {
		err = new(exec.PipedExec).
			Command("gh", "pr", "merge", prurl, "--squash", "-R", parentrepo).
			Run(os.Stdout, os.Stdout)
		if err != nil {
			return err
		}

		repo, org := GetRepoAndOrgName()
		if len(repo) > 0 {
			err = new(exec.PipedExec).
				Command("gh", "repo", "sync", org+slash+repo).
				Run(os.Stdout, os.Stdout)
		}

	}
	return err
}

func retrieveRepoNameFromUPL(prurl string) string {
	var strs []string = strings.Split(prurl, slash)
	if len(strs) < minRepoNameLength {
		return ""
	}
	res := ""
	lenstr := len(strs)
	for i := lenstr - minRepoNameLength; i < lenstr-2; i++ {
		if res == "" {
			res = strs[i]
		} else {
			res = res + slash + strs[i]
		}
	}
	return res
}

func prCheckAbsent(val *gchResponse) bool {
	return strings.Contains(val._stderr, nochecksmsg)
}

func prCheckSuccess(val *gchResponse) bool {
	ss := strings.Split(val._stdout, caret)
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
	waitTimer := time.NewTimer(prTimeWait * time.Second)
	fmt.Print(strw)
	for {
		select {
		case val, ok = <-c:
			fmt.Println("")
			if ok {
				return val
			}
			return &gchResponse{_err: errors.New(ErrSomethigWrong)}
		case <-waitTimer.C:
			fmt.Println("")
			return &gchResponse{_err: errors.New(ErrTimer40Sec)}
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
		stdout, stderr, err = new(exec.PipedExec).
			Command("gh", "pr", "checks", prurl, "--watch").
			RunToStrings()
	} else {
		stdout, stderr, err = new(exec.PipedExec).
			Command("gh", "pr", "checks", prurl, "--watch", "-R", parentrepo).
			RunToStrings()
	}
	c <- &gchResponse{stdout, stderr, err}
}

func GetCurrentBranchName() string {
	stdout, _, _ := new(exec.PipedExec).
		Command(git, branch).
		Command("sed", "-n", "/\\* /s///p").
		RunToStrings()
	return strings.TrimSpace(stdout)
}

// getRemotes shows list of names of all remotes
func getRemotes() []string {
	stdout, _, _ := new(exec.PipedExec).
		Command(git, "remote").
		RunToStrings()
	strs := strings.Split(stdout, caret)
	for i, str := range strs {
		if len(strings.TrimSpace(str)) == 0 {
			strs = append(strs[:i], strs[i+1:]...)
		}
	}
	return strs
}

// GetFilesForCommit shows list of file names, ready for commit
func GetFilesForCommit() []string {
	stdout, _, _ := new(exec.PipedExec).
		Command(git, "status", "-s").
		RunToStrings()
	ss := strings.Split(stdout, caret)
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
		branch = mainBrachName
	}
	errupstream := new(exec.PipedExec).
		Command(git, "push", "--set-upstream", repo, branch).
		Run(os.Stdout, os.Stdout)
	ExitIfError(errupstream)
}

// GetCommitFileSizes returns quantity of cmmited files and their total sizes
func GetCommitFileSizes() (totalSize int, quantity int) {
	totalSize = 0
	quantity = 0
	stdout, _, err := new(exec.PipedExec).
		Command(git, "status", "--porcelain").
		Command("gawk", "{if ($1 == \"??\") print $2}").
		RunToStrings()
	ExitIfError(err)
	files := strings.Split(stdout, caret)

	if len(files) == 0 {
		return
	}

	for _, file := range files {
		if len(file) > 0 {

			stdout, _, err = new(exec.PipedExec).
				Command("wc", "-c", file).
				Command("gawk", "{print $1}").
				RunToStrings()
			ExitIfError(err)

			strval := strings.TrimSpace(stdout)
			if strval != "" {
				sz, err := strconv.Atoi(strval)
				if err != nil {
					fmt.Println("Error during conversion of value: ", err.Error())
					return
				}
				totalSize += sz
				quantity++
			}
		}
	}
	return totalSize, quantity
}

func getGlobalHookFolder() string {
	stdout, _, err := new(exec.PipedExec).
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
	ExitIfError(err)

	filepath += "/pre-commit"
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

	_, _, err := new(exec.PipedExec).
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
		ExitIfError(err)
		path = rootUser.HomeDir
		path += "/.git/hooks"
		err = os.MkdirAll(path, os.ModePerm)
		ExitIfError(err)
	}

	// Set global hooks folder
	err = new(exec.PipedExec).
		Command(git, "config", "--global", "core.hookspath", path).
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)

	filepath := path + "/pre-commit"
	f := createOrOpenFile(filepath)
	f.Close()
	if !largFileHookExist(filepath) {
		fillPreCommitFile(filepath)
	}
}

func GetRootFolder() string {
	stdouts, _, err := new(exec.PipedExec).
		Command(git, "rev-parse", "--show-toplevel").
		RunToStrings()
	ExitIfError(err)
	return strings.TrimSpace(stdouts)
}

// SetLocalPreCommitHook - s.e.
func SetLocalPreCommitHook() {

	// Turn off globa1 hooks

	err := new(exec.PipedExec).
		Command(git, "config", "--global", "--get", "core.hookspath").
		Run(os.Stdout, os.Stdout)
	if nil == err {
		err = new(exec.PipedExec).
			Command(git, "config", "--global", "--unset", "core.hookspath").
			Run(os.Stdout, os.Stdout)
		ExitIfError(err)
	}
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
		ExitIfError(err)
		_, err = f.WriteString("#!/bin/bash\n")
	} else {
		f, err = os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, bashFilePerm)
	}
	ExitIfError(err)
	return f
}

func fillPreCommitFile(myfilepath string) {
	f := createOrOpenFile(myfilepath)
	defer f.Close()

	pathLaregFile := "https://raw.githubusercontent.com/untillpro/ci-action/master/scripts/large-file-hook.sh"

	dir := GetRootFolder()
	fname := "/.git/hooks/large-file-hook.sh"
	lf := dir + fname

	err := new(exec.PipedExec).
		Command("curl", "-s", "-o", lf, pathLaregFile).
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)

	hookcode := "\n#Here is large files commit prevent is added by [qs]\n"
	hookcode = hookcode + "bash " + lf + caret
	_, err = f.WriteString(hookcode)
	ExitIfError(err)

	err = new(exec.PipedExec).
		Command("chmod", "+x", myfilepath).
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)
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
		ExitIfError(errors.New("upstream is not set, Pull Request can not be done"))
	}

	err := new(exec.PipedExec).
		Command(git, "fetch", "upstream").
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)

	_, _, err = new(exec.PipedExec).
		Command(git, "diff", "--quiet", brName+"...upstream/"+mainbr).
		RunToStrings()

	return err != nil

}

func MergeFromUpstreamRebase() {
	mainbr := GetMainBranch()
	err := new(exec.PipedExec).
		Command(git, "fetch", "upstream").
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)

	err = new(exec.PipedExec).
		Command(git, "rebase", "upstream/"+mainbr).
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)

	brName := GetCurrentBranchName()
	err = new(exec.PipedExec).
		Command(git, push, "-f", origin, brName).
		Run(os.Stdout, os.Stdout)
	ExitIfError(err)
}

func GawkInstalled() bool {
	_, _, err := new(exec.PipedExec).
		Command("gawk", "--version").
		RunToStrings()
	return err == nil
}

// GHInstalled returns is gh utility installed
func GHInstalled() bool {
	_, _, err := new(exec.PipedExec).
		Command("gh", "--version").
		RunToStrings()
	return err == nil
}

// GHLoggedIn returns is gh logged in
func GHLoggedIn() bool {
	_, _, err := new(exec.PipedExec).
		Command("gh", "auth", "status").
		RunToStrings()
	return err == nil
}

func GetInstalledQSVersion() string {
	stdouts, stderr, err := new(exec.PipedExec).
		Command("go", "env", "GOPATH").
		RunToStrings()
	if err != nil {
		logger.Error("GetInstalledVersion error:", stderr)
	}

	gopath := strings.TrimSpace(stdouts)
	if len(gopath) == 0 {
		logger.Error("GetInstalledVersion error:", errors.New("GOPATH is not defined"))
	}
	qsexe := "qs"
	if runtime.GOOS == "windows" {
		qsexe = "qs.exe"
	}

	stdouts, stderr, err = new(exec.PipedExec).
		Command("go", "version", "-m", gopath+"/bin/"+qsexe).
		Command("grep", "-i", "-h", "mod.*github.com/untillpro/qs").
		Command("gawk", "{print $3}").
		RunToStrings()
	if err != nil {
		logger.Error("GetInstalledQSVersion error:", stderr)
	}
	return strings.TrimSpace(stdouts)
}

func GetLastQSVersion() string {
	stdouts, stderr, err := new(exec.PipedExec).
		Command("go", "list", "-m", "-versions", "github.com/untillpro/qs").
		RunToStrings()
	if err != nil {
		logger.Error("GetLastQSVersion error:", stderr)
	}

	arr := strings.Split(strings.TrimSpace(stdouts), oneSpace)
	if len(arr) == 0 {
		return ""
	}

	return arr[len(arr)-1]
}

func extractIntegerPrefix(input string) (string, error) {
	// Define the regular expression pattern
	pattern := `^\d+`
	re := regexp.MustCompile(pattern)

	// Find the match
	match := re.FindString(input)
	if match == "" {
		return "", fmt.Errorf("no integer found at the beginning of the string")
	}

	// Convert the matched string to an integer
	integerValue, err := strconv.Atoi(match)
	if err != nil {
		return "", fmt.Errorf("error converting string to integer: %v", err)
	}

	return strconv.Itoa(integerValue), nil
}

func issuenumExists(parentrepo string, issuenum string) bool {
	stdouts, _, err := new(exec.PipedExec).
		Command("gh", "issue", "develop", issuenum, "--list", "-R", parentrepo).
		Command("gawk", "{print $2}").
		RunToStrings()
	if (err == nil) && (len(stdouts) > minIssueNoteLength) {
		names := strings.Split(stdouts, caret)
		for _, name := range names {
			if strings.Contains(name, slash+issuenum+"-") {
				return true
			}
		}
	}
	return false
}

func GetIssueNumFromBranchName(parentrepo string, curbranch string) (issuenum string, ok bool) {

	tempissuenum, err := extractIntegerPrefix(curbranch)
	if tempissuenum == "" {
		return "", false
	}
	if err == nil {
		if issuenumExists(parentrepo, tempissuenum) {
			return tempissuenum, true
		}
	}

	stdouts, stderr, err := new(exec.PipedExec).
		Command("gh", "issue", "list", "-R", parentrepo).
		Command("gawk", "{print $1}").
		RunToStrings()
	if err != nil {
		logger.Error("GetIssueNumFromBranchName:", stderr)
		return "", false
	}
	issuenums := strings.Split(stdouts, caret)
	fmt.Println("Searching linked issue ")

	for _, issuenum := range issuenums {
		if len(issuenum) > 0 {
			fmt.Println("  Issue number: ", issuenum, "...")
			if issuenumExists(parentrepo, issuenum) {
				return issuenum, true
			}
		}
	}

	return "", false
}

func GetIssuePRTitle(issueNum string, parentrepo string) []string {
	name := GetIssueNameByNumber(issueNum, parentrepo)
	s := IssuePRTtilePrefix + oneSpace + name
	body := IssueSign + issueNum + oneSpace + name
	return []string{s, body}
}

func LinkIssueToMileStone(issueNum string, parentrepo string) {
	if issueNum == "" {
		return
	}
	if parentrepo == "" {
		return
	}
	strMilestones, _, err := new(exec.PipedExec).
		Command("gh", "api", "repos/"+parentrepo+"/milestones", "--jq", ".[] | .title").
		RunToStrings()
	if err != nil {
		fmt.Println("Link issue to mileStone error: ", err.Error())
		return
	}
	milestones := strings.Split(strMilestones, caret)
	// Sample date string in the "yyyy.mm.dd" format.
	dateString := "2006.01.02"
	// Get the current date and time.
	currentTime := time.Now()
	for _, milestone := range milestones {
		// Parse the input string into a time.Time value.
		t, err := time.Parse(dateString, milestone)
		if err == nil {
			if currentTime.Before(t) {
				// Next milestone is found
				err = new(exec.PipedExec).
					Command("gh", "issue", "edit", issueNum, "--milestone", milestone, "--repo", parentrepo).
					Run(os.Stdout, os.Stdout)
				ExitIfError(err)
				fmt.Println("Issue #" + issueNum + " added to milestone '" + milestone + "'")
				return
			}
		}
	}
}
