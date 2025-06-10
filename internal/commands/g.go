package commands

import "github.com/untillpro/qs/git"

func G() error {
	globalConfig()

	return git.Gui()
}
