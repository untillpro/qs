package commands

import (
	"github.com/untillpro/qs/git"
)

func D() error {
	globalConfig()

	return git.Download()
}
