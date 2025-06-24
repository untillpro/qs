package commands

import (
	"fmt"

	"github.com/untillpro/qs/internal/helper"
)

func Version() error {
	globalConfig()
	ver, err := helper.GetInstalledQSVersion()
	if err != nil {
		return err
	}
	fmt.Printf("qs version %s\n", ver)

	return nil
}
