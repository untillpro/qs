package commands

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"golang.org/x/mod/semver"
)

func Version() error {
	ver, err := GetInstalledQSVersion()
	if err != nil {
		return err
	}
	fmt.Printf("qs version %s\n", ver)

	return nil
}

func CheckQsVer() bool {
	installedVer, err := GetInstalledQSVersion()
	if err != nil {
		logger.Verbose(fmt.Sprintf("Error getting installed qs version: %v", err))

		return false
	}

	lastQSVersion := getLastQSVersion()
	if semver.Compare(installedVer, lastQSVersion) < 0 {
		fmt.Printf("Installed qs version %s is too old (last version is %s)\n", installedVer, lastQSVersion)
		fmt.Println("You can install last version with:")
		fmt.Println("-----------------------------------------")
		fmt.Println("go install github.com/untillpro/qs@latest")
		fmt.Println("-----------------------------------------")
		fmt.Print("Ignore it and continue with current version(y/n)?")
		var response string
		_, _ = fmt.Scanln(&response)

		return response == "y"
	}

	return true
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
	stdout, stderr, err := new(exec.PipedExec).
		Command("go", "list", "-m", "-versions", "github.com/untillpro/qs").
		RunToStrings()
	if err != nil {
		logger.Verbose(fmt.Sprintf("getLastQSVersion error: %v", stderr))
	}

	arr := strings.Split(strings.TrimSpace(stdout), " ")
	if len(arr) == 0 {
		return ""
	}

	return arr[len(arr)-1]
}
