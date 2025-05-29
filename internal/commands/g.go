package commands

import "github.com/untillpro/qs/git"

func G() {
	globalConfig()
	git.Gui()
}
