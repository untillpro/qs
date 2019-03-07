package git

import (
	"fmt"
	"io/ioutil"
	"os"

	coreos "github.com/coreos/go-semver/semver"
	"github.com/untillpro/gochips"
	"github.com/untillpro/qs/vcs"
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
	dat, err := ioutil.ReadFile("version")
	gochips.ExitIfError(err, "Error reading file 'version'")
	sdat := string(dat)
	currentVersion := *coreos.New(sdat)
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
	gochips.ExitIfError(ioutil.WriteFile("version", []byte(targetVersion.String()), 0644))

	// *************************************************
	gochips.Doing("Commiting target version")
	{
		params := []string{"commit", "-a", "-m", "#scm-ver " + targetVersion.String()}
		err = new(gochips.PipedExec).
			Command("git", params...).
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
	}

	// *************************************************
	gochips.Doing("Tagging")
	{
		params := []string{"tag", "v" + targetVersion.String()}
		err = new(gochips.PipedExec).
			Command("git", params...).
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
	}

	// *************************************************
	gochips.Doing("Bumping version")
	newVersion := currentVersion
	{
		newVersion.Minor++
		gochips.ExitIfError(ioutil.WriteFile("version", []byte(newVersion.String()), 0644))
	}

	// *************************************************
	gochips.Doing("Commiting new version")
	{
		params := []string{"commit", "-a", "-m", "#scm-ver " + newVersion.String()}
		err = new(gochips.PipedExec).
			Command("git", params...).
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
	}

	// *************************************************
	gochips.Doing("Pushing to origin")
	{
		params := []string{"push", "origin"}
		err = new(gochips.PipedExec).
			Command("git", params...).
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
	}

	// *************************************************
	gochips.Doing("Pushing tags to origin")
	{
		params := []string{"push", "origin", "--tags"}
		err = new(gochips.PipedExec).
			Command("git", params...).
			Run(os.Stdout, os.Stdout)
		gochips.ExitIfError(err)
	}

}

// Upload upload sources to git repo
func Upload(cfg vcs.CfgUpload) {
	new(gochips.PipedExec).
		Command("git", "add", ".").
		Run(os.Stdout, os.Stdout)

	params := []string{"commit", "-a"}
	for _, m := range cfg.Message {
		params = append(params, "-m", m)
	}

	new(gochips.PipedExec).
		Command("git", params...).
		Run(os.Stdout, os.Stdout)

	new(gochips.PipedExec).
		Command("git", "push").
		Run(os.Stdout, os.Stdout)
}

// Download sources from git repo
func Download(cfg vcs.CfgDownload) {
	new(gochips.PipedExec).
		Command("git", "pull").
		Run(os.Stdout, os.Stdout)
}

// Gui shows gui
func Gui() {
	new(gochips.PipedExec).
		Command("git", "gui").
		Run(os.Stdout, os.Stdout)
}
