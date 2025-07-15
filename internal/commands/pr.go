package commands

import (
	"github.com/untillpro/qs/gitcmds"
)

func Pr(wd string, needDraft bool) error {
	if _, err := gitcmds.CheckIfGitRepo(wd); err != nil {
		return err
	}

	return gitcmds.Pr(wd, needDraft)
}
