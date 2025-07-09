package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/untillpro/qs/gitcmds"
	"github.com/untillpro/qs/internal/helper"
)

func Pr(wd string, needDraft bool) error {
	globalConfig()
	if _, err := gitcmds.CheckIfGitRepo(wd); err != nil {
		return err
	}

	skipQsVerCheck, _ := strconv.ParseBool(os.Getenv(EnvSkipQsVersionCheck))
	if !skipQsVerCheck && !helper.CheckQsVer() {
		return fmt.Errorf("qs version check failed")
	}
	if !helper.CheckGH() {
		return fmt.Errorf("GitHub CLI check failed")
	}

	return gitcmds.Pr(wd, needDraft)
}
