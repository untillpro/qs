package main

import (
	"context"
	"fmt"
	"os"

	"github.com/untillpro/qs/internal/cmdproc"
)

func main() {
	checkPrerequisites()

	if _, err := cmdproc.ExecRootCmd(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func checkPrerequisites() {
	requiredCommands := []string{"grep", "sed", "jq", "gawk", "wc", "curl", "chmod"}
	if err := cmdproc.CheckCommands(requiredCommands); err != nil {
		fmt.Println(" ")
		fmt.Println(err)
		fmt.Println("See https://github.com/untillpro/qs?tab=readme-ov-file#git")

		os.Exit(1)
	}
}
