package commands

import (
	"github.com/untillpro/qs/gitcmds"
)

func D(wd string) error {
	globalConfig()

	return gitcmds.Download(wd)
}
