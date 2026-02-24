package main

import (
	"context"
	"os"

	"github.com/untillpro/qs/internal/cmdproc"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

func main() {
	if _, err := cmdproc.ExecRootCmd(context.Background(), os.Args); err != nil {
		logger.Verbose(err)

		os.Exit(1)
	}
}
