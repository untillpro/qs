package helper

import (
	"fmt"

	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/git"
)

func CheckGH() bool {
	if !git.GHInstalled() {
		fmt.Print("\nGithub cli utility 'gh' is not installed.\nTo install visit page https://cli.github.com/\n")

		return false
	}
	if !git.GHLoggedIn() {
		fmt.Print("\nGH utility is not logged in\n")

		return false
	}

	return true
}

func CheckQsVer() bool {
	installedVer, err := git.GetInstalledQSVersion()
	if err != nil {
		logger.Verbose("Error getting installed qs version: %s\n", err)

		return false
	}
	lastQSVersion := git.GetLastQSVersion()

	if installedVer != lastQSVersion {
		fmt.Printf("Installed qs version %s is too old (last version is %s)\n", installedVer, lastQSVersion)
		fmt.Println("You can install last version with:")
		fmt.Println("-----------------------------------------")
		fmt.Println("go install github.com/untillpro/qs@latest")
		fmt.Println("-----------------------------------------")
		fmt.Print("Ignore it and continue with current version(y/n)?")
		var response string
		_, _ = fmt.Scanln(&response)

		return response == pushYes
	}

	return true
}
