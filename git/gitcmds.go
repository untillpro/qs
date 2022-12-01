package git

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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
	nochecksmsg           = "no checks reported"
	msgWaitingPR          = "Waiting PR checks.."
	msgPRCheckNotFoundYet = "..not found yet"
	msgPRCheckNotFound    = "No checks for PR found, merge without checks"

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
)

type gchResponse struct {
	_stdout string
	_stderr string
	_err    error
}

// ChangedFilesExist s.e.
func ChangedFilesExist() (uncommitedFiles string, exist bool) {
	stdouts, _, err := new(gochips.PipedExec).
		Command("git", "status", "-s").
		RunToStrings()
	gochips.ExitIfError(err)
	uncommitedFiles = strings.TrimSpace(stdouts)
	exist = len(uncommitedFiles) > 0
	return uncommitedFiles, exist
}

// Status shows git repo status
func Status(cfg vcs.CfgStatus) {
	err := new(gochips.PipedExec).
		Command("git", "remote", "-v").
		Command("grep", fetch).
		Command("sed", "s/(fetch)//").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
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

	err = new(gochips.PipedExec).
		Command(git, params...).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	err = new(gochips.PipedExec).
		Command(git, push).
		Run(os.Stdout, os.Stdout)
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

// GetRepoAndOrgName - from .git/config
func GetRepoAndOrgName() (repo string, org string) {
	stdouts, _, err := new(gochips.PipedExec).
		Command(git, "config", "--local", "remote.origin.url").
		RunToStrings()
	if err != nil {
		return "", ""
	}
	repourl := strings.TrimSpace(stdouts)
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
	err := new(gochips.PipedExec).
		Command(git, "stash", "pop").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
}

func GetMainBranch() string {
	stdouts, _, err := new(gochips.PipedExec).
		Command(git, branch, "-r").
		RunToStrings()
	if err == nil {
		brlistraw := strings.Split(stdouts, "\n")
		for _, branchstr := range brlistraw {
			arr := strings.Split(branchstr, "->")
			if len(arr) > 1 {
				if strings.Contains(arr[1], "/") {
					str := strings.Split(arr[1], slash)
					if len(str) > 0 {
						return strings.TrimSpace(str[len(str)-1])
					}
				}
			}
		}
	}
	return ""
}

func getUserName() string {
	stdouts, _, err := new(gochips.PipedExec).
		Command(git, "config", "--global", "user.name").
		RunToStrings()
	gochips.ExitIfError(err)
	return strings.TrimSpace(stdouts)
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
	err = new(gochips.PipedExec).
		Command(git, fetch, "origin").
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
	err = new(gochips.PipedExec).
		Command(git, branch, "--set-upstream-to", "origin/"+mainbranch, mainbranch).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
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
		Command(git, "checkout", mainbrach).
		Run(os.Stdout, os.Stdout)
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
	err = new(gochips.PipedExec).
		Command(git, push, "-u", origin, branch).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)

	addNotes(comments)

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
	// Remove all existing Notes
	if notesObsj, ok := GetNotesObj(); ok {
		for _, notesObj := range notesObsj {
			str := strings.TrimSpace(notesObj)
			if len(str) > 0 {
				err := new(gochips.PipedExec).
					Command(git, "notes", "remove", str).
					Run(os.Stdout, os.Stdout)
				gochips.ExitIfError(err)
			}
		}
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
	noteobjs, ok := GetNotesObj()
	if !ok {
		return nil, false
	}
	obj := ""
	if len(noteobjs) > 0 {
		obj = noteobjs[len(noteobjs)-1]
	}
	stdouts, _, err := new(gochips.PipedExec).
		Command(git, "notes", "show", obj).
		RunToStrings()
	gochips.ExitIfError(err)
	notes = strings.Split(strings.ReplaceAll(stdouts, "\r\n", "\n"), "\n")
	return notes, true
}

// GetNotesObj s.e.
func GetNotesObj() (notes []string, result bool) {
	stdouts, _, err := new(gochips.PipedExec).
		Command(git, "notes", "list").
		RunToStrings()
	gochips.ExitIfError(err)
	stdouts = strings.TrimSpace(stdouts)
	if len(stdouts) == 0 {
		return nil, false
	}
	noteLines := strings.Split(strings.ReplaceAll(stdouts, "\r\n", "\n"), "\n")
	if len(noteLines) == 0 {
		return nil, false
	}
	notes = make([]string, len(noteLines))
	for _, noteLine := range noteLines {
		noteLine = strings.TrimSpace(noteLine)
		ns := strings.Split(noteLine, " ")
		if len(ns) == 2 {
			s := strings.TrimSpace(ns[1])
			notes = append(notes, s)
		}
	}
	return notes, true
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
		Command(git, "checkout", mainbrach).
		Run(os.Stdout, os.Stdout)
	gochips.ExitIfError(err)
	err = new(gochips.PipedExec).
		Command(git, "checkout", "-b", branch).
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

// GetParentRepoName - parent repo of forked
func GetParentRepoName() (name string) {
	repo, org := GetRepoAndOrgName()
	stdouts, _, err := new(gochips.PipedExec).
		Command("gh", "api", "repos/"+org+"/"+repo, "--jq", ".parent.full_name").
		RunToStrings()
	gochips.ExitIfError(err)
	name = strings.TrimSpace(stdouts)
	return
}

// IsBranchInMain Is my branch in main org?
func IsBranchInMain() bool {
	repo, org := GetRepoAndOrgName()
	parent := GetParentRepoName()
	return parent == org+"/"+repo
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
		Command("grep", ": gone]").
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
			} else {
				if len(note) == 0 {
					note = s
				} else {
					note = note + " " + s
				}
			}
		}
	}
	return note, url
}

// MakePR s.e.
func MakePR(notes []string) (err error) {
	if len(notes) == 0 {
		return errors.New(errMsgPRNotesImpossible)
	}
	var strnotes string
	var url string
	strnotes, url = GetNoteAndURL(notes)

	body := strnotes
	if len(url) > 0 {
		body = body + "\n" + url
	}
	strbody := fmt.Sprintln(body)
	parentrepo := GetParentRepoName()
	err = new(gochips.PipedExec).
		Command("gh", "pr", "create", "-t", strnotes, "-b", strbody, "-R", parentrepo).
		Run(os.Stdout, os.Stdout)
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
			fmt.Println("repo:", repo)
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
	return stdout
}
