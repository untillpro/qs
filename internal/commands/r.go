package commands

import "github.com/untillpro/qs/git"

func R() error {
	globalConfig()

	return git.Release()
}
