package utils

import (
	"os"
	"testing"
)

func TestPipedExec_Start(t *testing.T) {
	new(PipedExec).
		Command("ls").Wd("/").
		Command("grep", "eclipse").
		Run(os.Stdout, os.Stdout)
}
