package commands

import (
	"fmt"

	"github.com/untillpro/qs/git"
)

func Version() error {
	globalConfig()
	ver, err := git.GetInstalledQSVersion()
	if err != nil {
		return err
	}
	fmt.Printf("qs version %s\n", ver)

	return nil
}
