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
	mm                  = "-m"
	slash               = "/"
	git                 = "git"
	push                = "push"
	origin              = "origin"
	repoNotFound        = "git repo name not found"
	userNotFound        = "git user name not found"
	errAlreadyForkedMsg = "You are in fork already\nExecute 'qs dev [branch name]' to create dev branch"
)

// Status shows git repo status
func Status(cfg vcs.CfgStatus) {
	err := new(gochips.PipedExec).
		Command("git", "remote", "-v").
		Command("grep", "fetch").
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
		params := []string{"commit", "-a", mm, "#scm-ver " + targetVersion.String()}
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
		params := []string{"tag", mm, "Version " + tagName + " of " + n.Format("2006/01/02 15:04:05"), tagName}
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
		params := []string{"commit", "-a", mm, "#scm-ver " + newVersion.String()}
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
		params = append(params, mm, m)
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

// GetRepoName  -from .git/config
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

// GetRemoteUpstreamURL - from .git/config
func GetRemoteUpstreamURL() string {
	stdouts, _, err := new(gochips.PipedExec).
		Command(git, "config", "--local", "remote.upstream.url").
		RunToStrings()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdouts)
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
	err = new(gochips.PipedExec).
		Command("gh", "repo", "fork", org+slash+repo, "--clone=false").
		Run(os.Stdout, os.Stdout)
	if err != nil {
		return repo, err
	}

	return repo, nil
}

func getMainBranch() string {
	stdouts, _, err := new(gochips.PipedExec).
		Command(git, "symbolic-ref", "--short", "refs/remotes/upstream/HEAD").
		RunToStrings()
	if err == nil {
		str := strings.Split(stdouts, slash)
		if len(str) > 0 {
			return strings.TrimSpace(str[len(str)-1])
		}
	}
	stdouts, _, err = new(gochips.PipedExec).
		Command(git, "symbolic-ref", "--short", "HEAD").
		RunToStrings()
	gochips.ExitIfError(err)
	return strings.TrimSpace(stdouts)
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

	branch := getMainBranch()
	new(gochips.PipedExec).
		Command(git, "remote", "rename", "origin", "upstream").
		Run(os.Stdout, os.Stdout)
	new(gochips.PipedExec).
		Command(git, "remote", "add", "origin", "https://github.com/"+user+slash+repo).
		Run(os.Stdout, os.Stdout)
	new(gochips.PipedExec).
		Command(git, "fetch", "origin").
		Run(os.Stdout, os.Stdout)
	new(gochips.PipedExec).
		Command(git, "branch", "--set-upstream-to", "origin/"+branch, branch).
		Run(os.Stdout, os.Stdout)
}

// Dev branch
func Dev(branch string) {
	mainbrach := getMainBranch()
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
}
