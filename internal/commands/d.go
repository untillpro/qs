package commands

import (
	"github.com/untillpro/qs/gitcmds"
)

func D(wd string) error {
	return gitcmds.Download(wd)
}
