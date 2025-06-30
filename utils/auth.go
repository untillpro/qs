package utils

import (
	"fmt"
	"os"
	"os/exec"
)

func GhAuthLogin(token string) error {
	// Connect stdin to pass the token to the gh process
	cmd := exec.Command("gh", "auth", "login", "--with-token")

	// Important: Connect stdout and stderr too
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	// Write token immediately

	if _, err := stdin.Write([]byte(token)); err != nil {
		return fmt.Errorf("failed to write token to stdin: %w", err)
	}

	// Start the command first!
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start gh auth login: %w", err)
	}

	return stdin.Close() // IMPORTANT: Close stdin immediately after writing
}
