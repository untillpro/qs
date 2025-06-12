package commands

import "github.com/untillpro/qs/gitcmds"

func R(wd string) error {
	globalConfig()

	return gitcmds.Release(wd)
}
