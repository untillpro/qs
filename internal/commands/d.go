package commands

import (
	"github.com/untillpro/qs/git"
	"github.com/untillpro/qs/vcs"
)

func D(cfg vcs.CfgDownload) {
	globalConfig()
	git.Download(cfg)
}
