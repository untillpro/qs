package commands

import "github.com/untillpro/qs/gitcmds"

func G(wd string) error {
	globalConfig()

	return gitcmds.Gui(wd)
}
