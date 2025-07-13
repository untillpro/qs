package commands

import "github.com/untillpro/qs/gitcmds"

func G(wd string) error {
	return gitcmds.Gui(wd)
}
