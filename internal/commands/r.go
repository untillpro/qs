package commands

import "github.com/untillpro/qs/git"

func R() {
	globalConfig()
	git.Release()
}
