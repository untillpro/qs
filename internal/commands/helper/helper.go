package helper

import (
	"fmt"

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

func CheckQSver() bool {
	installedver := git.GetInstalledQSVersion()
	lastver := git.GetLastQSVersion()

	if installedver != lastver {
		fmt.Printf("Installed qs version %s is too old (last version is %s)\n", installedver, lastver)
		fmt.Println("You can install last version with:")
		fmt.Println("-----------------------------------------")
		fmt.Println("go install github.com/untillpro/qs@latest")
		fmt.Println("-----------------------------------------")
		fmt.Print("Ignore it and continue with current version(y/n)?")
		var response string
		fmt.Scanln(&response)
		return response == pushYes
	}
	return true
}
