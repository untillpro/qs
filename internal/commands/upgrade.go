package commands

import "fmt"

func Upgrade() error {
	fmt.Println("\ngo install github.com/untillpro/qs@latest")

	return nil
}
