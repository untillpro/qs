package main

import (
	"context"
	"os"

	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/internal/cmdproc"
)

func main() {
	if _, err := cmdproc.ExecRootCmd(context.Background(), os.Args); err != nil {
		logger.Verbose(err)

		os.Exit(1)
	}
}
