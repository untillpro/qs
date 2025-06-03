package commands

import (
	"fmt"

	"github.com/untillpro/qs/git"
)

func Version() {
	globalConfig()
	ver := git.GetInstalledQSVersion()
	fmt.Printf("qs version %s\n", ver)
}
