package main

import (
	"fmt"
	"os"
)

func main() {

	requiredCommands := []string{"grep", "sed", "jq", "gawk", "wc", "curl", "chmod"}
	err := checkCommands(requiredCommands)
	if err != nil {
		fmt.Println(" ")
		fmt.Println(err)
		fmt.Println("See https://github.com/untillpro/qs?tab=readme-ov-file#git")
		os.Exit(1)
	}
	cmdproc := buildCommandProcessor().
		addUpdateCmd().
		addDownloadCmd().
		addReleaseCmd().
		addGUICmd().
		addForkBranch().
		addDevBranch().
		addPr().
		addUpgrade().
		addVersion()
	cmdproc.Execute()
}
