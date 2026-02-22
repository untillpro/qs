package commands

import (
	"github.com/untillpro/qs/gitcmds"
)

func Pr(wd string, needDraft bool) error {
	return gitcmds.Pr(wd, needDraft)
}
