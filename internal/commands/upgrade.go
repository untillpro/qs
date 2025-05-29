package commands

import "fmt"

func Upgrade() {
	globalConfig()
	fmt.Println("\ngo install github.com/untillpro/qs@latest")
}
