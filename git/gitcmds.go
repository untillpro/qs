package git

import (
	"fmt"
	"io/ioutil"
	"os"

	coreos "github.com/coreos/go-semver/semver"
	u "github.com/untillpro/qs/utils"
	"github.com/untillpro/qs/vcs"
)

// Status shows git repo status
func Status(cfg vcs.CfgStatus) {
	err := new(u.PipedExec).
		Command("git", "remote", "-v").
		Command("grep", "fetch").
		Command("sed", "s/(fetch)//").
		Run(os.Stdout, os.Stdout)
	if nil != err {
		return
	}
	new(u.PipedExec).
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
	u.Doing("Pulling")
	err := new(u.PipedExec).
		Command("git", "pull").
		Run(os.Stdout, os.Stdout)
	u.ExitIfError(err)

	// *************************************************
	u.Doing("Reading current version")
	dat, err := ioutil.ReadFile("version")
	u.ExitIfError(err, "Error reading file 'version'")
	sdat := string(dat)
	currentVersion := *coreos.New(sdat)
	u.Assert(len(currentVersion.PreRelease) > 0, "pre-release part of version does not exist: "+currentVersion.String())

	// Calculate target version

	targetVersion := currentVersion
	targetVersion.PreRelease = ""

	fmt.Printf("Version %v will be tagged, bumped and pushed, agree? [y]", targetVersion)
	var response string
	fmt.Scanln(&response)
	u.Assert(response == "y")

	// *************************************************
	u.Doing("Updating 'version' file")
	u.ExitIfError(ioutil.WriteFile("version", []byte(targetVersion.String()), 0644))

	// *************************************************
	u.Doing("Commiting target version")
	{
		params := []string{"commit", "-a", "-m", "#scm-ver " + targetVersion.String()}
		err = new(u.PipedExec).
			Command("git", params...).
			Run(os.Stdout, os.Stdout)
		u.ExitIfError(err)
	}

	// *************************************************
	u.Doing("Tagging")
	{
		params := []string{"tag", "v" + targetVersion.String()}
		err = new(u.PipedExec).
			Command("git", params...).
			Run(os.Stdout, os.Stdout)
		u.ExitIfError(err)
	}

	// *************************************************
	u.Doing("Bumping version")
	newVersion := currentVersion
	{
		newVersion.Minor++
		u.ExitIfError(ioutil.WriteFile("version", []byte(newVersion.String()), 0644))
	}

	// *************************************************
	u.Doing("Commiting new version")
	{
		params := []string{"commit", "-a", "-m", "#scm-ver " + newVersion.String()}
		err = new(u.PipedExec).
			Command("git", params...).
			Run(os.Stdout, os.Stdout)
		u.ExitIfError(err)
	}

	// *************************************************
	u.Doing("Pushing to origin")
	{
		params := []string{"push", "origin"}
		err = new(u.PipedExec).
			Command("git", params...).
			Run(os.Stdout, os.Stdout)
		u.ExitIfError(err)
	}

	// *************************************************
	u.Doing("Pushing tags to origin")
	{
		params := []string{"push", "origin", "--tags"}
		err = new(u.PipedExec).
			Command("git", params...).
			Run(os.Stdout, os.Stdout)
		u.ExitIfError(err)
	}

}

// Upload upload sources to git repo
func Upload(cfg vcs.CfgUpload) {
	new(u.PipedExec).
		Command("git", "add", ".").
		Run(os.Stdout, os.Stdout)

	params := []string{"commit", "-a"}
	for _, m := range cfg.Message {
		params = append(params, "-m", m)
	}

	new(u.PipedExec).
		Command("git", params...).
		Run(os.Stdout, os.Stdout)

	new(u.PipedExec).
		Command("git", "push").
		Run(os.Stdout, os.Stdout)
}

// Download sources from git repo
func Download(cfg vcs.CfgDownload) {
	new(u.PipedExec).
		Command("git", "pull").
		Run(os.Stdout, os.Stdout)
}

// Gui shows gui
func Gui() {
	new(u.PipedExec).
		Command("git", "gui").
		Run(os.Stdout, os.Stdout)
}
