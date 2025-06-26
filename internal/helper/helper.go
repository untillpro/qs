package helper

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
)

func IsTest() bool {
	return testing.Testing()
}

func Delay() {
	time.Sleep(1000 * time.Millisecond)
}

func CheckGH() bool {
	if !ghInstalled() {
		fmt.Print("\nGithub cli utility 'gh' is not installed.\nTo install visit page https://cli.github.com/\n")

		return false
	}
	if !ghLoggedIn() {
		fmt.Print("\nGH utility is not logged in\n")

		return false
	}

	return true
}

func CheckQsVer() bool {
	installedVer, err := GetInstalledQSVersion()
	if err != nil {
		logger.Error("Error getting installed qs version: %s\n", err)

		return false
	}
	lastQSVersion := getLastQSVersion()

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

// ghInstalled returns is gh utility installed
func ghInstalled() bool {
	_, _, err := new(exec.PipedExec).
		Command("gh", "--version").
		RunToStrings()
	return err == nil
}

// ghLoggedIn returns is gh logged in
func ghLoggedIn() bool {
	_, _, err := new(exec.PipedExec).
		Command("gh", "auth", "status").
		RunToStrings()
	return err == nil
}

func GetInstalledQSVersion() (string, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command("go", "env", "GOPATH").
		RunToStrings()
	if err != nil {
		return "", fmt.Errorf("GetInstalledVersion error: %s", stderr)
	}

	gopath := strings.TrimSpace(stdout)
	if len(gopath) == 0 {
		return "", errors.New("GetInstalledVersion error: \"GOPATH is not defined\"")
	}
	qsExe := "qs"
	if runtime.GOOS == "windows" {
		qsExe = "qs.exe"
	}

	stdout, stderr, err = new(exec.PipedExec).
		Command("go", "version", "-m", gopath+"/bin/"+qsExe).
		Command("grep", "-i", "-h", "mod.*github.com/untillpro/qs").
		Command("gawk", "{print $3}").
		RunToStrings()
	if err != nil {
		return "", fmt.Errorf("GetInstalledQSVersion error: %s", stderr)
	}

	return strings.TrimSpace(stdout), nil
}

func getLastQSVersion() string {
	stdouts, stderr, err := new(exec.PipedExec).
		Command("go", "list", "-m", "-versions", "github.com/untillpro/qs").
		RunToStrings()
	if err != nil {
		logger.Error("getLastQSVersion error:", stderr)
	}

	arr := strings.Split(strings.TrimSpace(stdouts), oneSpace)
	if len(arr) == 0 {
		return ""
	}

	return arr[len(arr)-1]
}
