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
	mimm                       = "-m"
	slash                      = "/"
	git                        = "git"
	push                       = "push"
	fetch                      = "fetch"
	branch                     = "branch"
	checkout                   = "checkout"
	origin                     = "origin"
	repoNotFound               = "git repo name not found"
	stashedfiles               = "Stashed files"
	userNotFound               = "git user name not found"
	errAlreadyForkedMsg        = "You are in fork already\nExecute 'qs dev [branch name]' to create dev branch"
	errMsgPRNotesImpossible    = "Pull request without comments is impossible."
	errMsgPRMerge              = "URL of PR is needed"
	errMsgPRBadFormat          = "Pull request URL has bad format"
	errMsgPRBad                = "Pull request URL is incorrect"
	errMsgPRParent             = "Can not find parent repo for PR"
	errMsgCanNotFindBranchName = "Can not find branch name"
	strLen                     = 1024
)

func ChangedFilesExist() (string, bool) {
	stdouts, _, err := new(gochips.PipedExec).
		Command("git", "status", "-s").
		RunToStrings()
	gochips.ExitIfError(err)
	str := strings.TrimSpace(stdouts)
	return str, len(str) > 0
}

// Status shows git repo status
func Status(cfg vcs.CfgStatus) {
	err := new(gochips.PipedExec).
		Command("git", "remote", "-v").
		Command("grep", fetch).
		Command("sed", "s/(fetch)//").
		Run(os.Stdout, os.Stdout)
	if nil != err {
		return
	}
	new(gochips.PipedExec).
		Command("git", "status", "-s", "-b", "-uall").
		Run(os.Stdout, os.Stdout)
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
	new(gochips.PipedExec).
		Command(git, "add", ".").
		Run(os.Stdout, os.Stdout)

	params := []string{"commit", "-a"}
	for _, m := range cfg.Message {
		params = append(params, mimm, m)
	}

	new(gochips.PipedExec).
		Command(git, params...).
		Run(os.Stdout, os.Stdout)

	new(gochips.PipedExec).
		Command(git, push).
		Run(os.Stdout, os.Stdout)
}

// Download sources from git repo
func Download(cfg vcs.CfgDownload) {
	new(gochips.PipedExec).
		Command(git, "pull").
		Run(os.Stdout, os.Stdout)
}

// Gui shows gui
func Gui() {
	new(gochips.PipedExec).
		Command(git, "gui").
		Run(os.Stdout, os.Stdout)
}

// GetRepoAndOrgName - from .git/config
func GetRepoAndOrgName() (repo string, org string) {
	stdouts, _, err := new(gochips.PipedExec).
		Command(git, "config", "--local", "remote.origin.url").
		RunToStrings()
	gochips.ExitIfError(err)
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
	var chExist bool
	_, chExist = ChangedFilesExist()
	if chExist {
		new(gochips.PipedExec).
			Command(git, "add", ".").
			Run(os.Stdout, os.Stdout)
		new(gochips.PipedExec).
			Command(git, "stash").
			Run(os.Stdout, os.Stdout)
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
	new(gochips.PipedExec).
		Command(git, "stash", "pop").
		Run(os.Stdout, os.Stdout)
}

func getMainBranch() string {
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

	mainbranch := getMainBranch()
	new(gochips.PipedExec).
		Command(git, "remote", "rename", "origin", "upstream").
		Run(os.Stdout, os.Stdout)
	new(gochips.PipedExec).
		Command(git, "remote", "add", "origin", "https://github.com/"+user+slash+repo).
		Run(os.Stdout, os.Stdout)
	new(gochips.PipedExec).
		Command(git, fetch, "origin").
		Run(os.Stdout, os.Stdout)
	new(gochips.PipedExec).
		Command(git, branch, "--set-upstream-to", "origin/"+mainbranch, mainbranch).
		Run(os.Stdout, os.Stdout)
}

// Dev branch
func Dev(branch string, comments []string) {
	mainbrach := getMainBranch()
	var chExist bool
	_, chExist = ChangedFilesExist()
	if chExist {
		new(gochips.PipedExec).
			Command(git, "add", ".").
			Run(os.Stdout, os.Stdout)
		new(gochips.PipedExec).
			Command(git, "stash").
			Run(os.Stdout, os.Stdout)
	}
	new(gochips.PipedExec).
		Command(git, "checkout", mainbrach).
		Run(os.Stdout, os.Stdout)
	new(gochips.PipedExec).
		Command(git, "pull", "-p", "upstream", mainbrach).
		Run(os.Stdout, os.Stdout)
	new(gochips.PipedExec).
		Command(git, push).
		Run(os.Stdout, os.Stdout)
	new(gochips.PipedExec).
		Command(git, "checkout", "-b", branch).
		Run(os.Stdout, os.Stdout)
	new(gochips.PipedExec).
		Command(git, push, "-u", origin, branch).
		Run(os.Stdout, os.Stdout)

	addNotes(comments)

	if chExist {
		new(gochips.PipedExec).
			Command(git, "stash", "pop").
			Run(os.Stdout, os.Stdout)
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
				new(gochips.PipedExec).
					Command(git, "notes", "remove", str).
					Run(os.Stdout, os.Stdout)
			}
		}
	}
	// Add new Notes
	for _, s := range comments {
		str := strings.TrimSpace(s)
		if len(str) > 0 {
			new(gochips.PipedExec).
				Command(git, "notes", "append", "-m", s).
				Run(os.Stdout, os.Stdout)
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
	fmt.Println("stdouts:", len(stdouts))
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
	mainbrach := getMainBranch()
	var chExist bool
	_, chExist = ChangedFilesExist()
	if chExist {
		new(gochips.PipedExec).
			Command(git, "add", ".").
			Run(os.Stdout, os.Stdout)
		new(gochips.PipedExec).
			Command(git, "stash").
			Run(os.Stdout, os.Stdout)
	}
	new(gochips.PipedExec).
		Command(git, "checkout", mainbrach).
		Run(os.Stdout, os.Stdout)
	new(gochips.PipedExec).
		Command(git, "checkout", "-b", branch).
		Run(os.Stdout, os.Stdout)

	addNotes(comments)

	new(gochips.PipedExec).
		Command(git, push, "-u", origin, branch).
		Run(os.Stdout, os.Stdout)
	if chExist {
		new(gochips.PipedExec).
			Command(git, "stash", "pop").
			Run(os.Stdout, os.Stdout)
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

//IsBranchInMain Is my branch in main org?
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

	stdouts, _, err = new(gochips.PipedExec).
		Command(git, branch, "-r").
		RunToStrings()
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
	mainbranch := getMainBranch()
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

// MakePR s.e.
func MakePR(notes []string) (err error) {
	if len(notes) == 0 {
		return errors.New(errMsgPRNotesImpossible)
	}
	var strnotes string
	var url string
	fmt.Println("notes:  --------------- ", len(notes))
	for _, s := range notes {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			if strings.Contains(s, "https") {
				url = s
			} else {
				if len(strnotes) == 0 {
					strnotes = s
				} else {
					strnotes = strnotes + " " + s
				}
			}
		}
	}

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

// MakePRMerge s.e.
func MakePRMerge(prurl string) (err error) {
	if len(prurl) == 0 {
		return errors.New(errMsgPRMerge)
	}
	if !strings.Contains(prurl, "https") {
		return errors.New(errMsgPRBadFormat)
	}
	parentrepo := GetParentRepoName()
	var stderr string
	if len(parentrepo) == 0 {
		_, stderr, _ = new(gochips.PipedExec).
			Command("gh", "pr", "checks", prurl).
			RunToStrings()
	} else {
		_, stderr, _ = new(gochips.PipedExec).
			Command("gh", "pr", "checks", prurl, "-R", parentrepo).
			RunToStrings()
	}

	if strings.Contains(stderr, "not resolve") || strings.Contains(stderr, "no pull requests") {
		return errors.New(errMsgPRBad)
	}

	if len(parentrepo) == 0 {
		err = new(gochips.PipedExec).
			Command("gh", "pr", "merge", prurl, "--squash").
			Run(os.Stdout, os.Stdout)
	} else {
		err = new(gochips.PipedExec).
			Command("gh", "pr", "merge", prurl, "--squash", "-R", parentrepo).
			Run(os.Stdout, os.Stdout)
	}
	if err != nil {
		return err
	}

	repo, org := GetRepoAndOrgName()
	mainbranch := getMainBranch()
	err = new(gochips.PipedExec).
		Command("gh", "repo", "sync", repo+"/"+org, mainbranch).
		Run(os.Stdout, os.Stdout)

	return err
}
