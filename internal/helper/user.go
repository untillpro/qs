package helper

import (
	"strings"

	"github.com/untillpro/goutils/exec"
)

// GetUserEmail - github user email
func GetUserEmail() (string, error) {
	var stdout string
	err := Retry(func() error {
		var apiErr error
		stdout, _, apiErr = new(exec.PipedExec).
			Command("gh", "api", "user", "--jq", ".email").
			RunToStrings()
		return apiErr
	})

	return strings.TrimSpace(stdout), err
}
