package commands

import (
	"fmt"

	"github.com/untillpro/qs/gitcmds"
)

func Version() error {
	globalConfig()
	ver, err := gitcmds.GetInstalledQSVersion()
	if err != nil {
		return err
	}
	fmt.Printf("qs version %s\n", ver)

	return nil
}
