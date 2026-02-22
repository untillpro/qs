package cmdproc

import "github.com/untillpro/qs/internal/commands"

var (
	requiredBashCommands = []string{"grep", "sed", "jq", "gawk", "wc", "curl", "chmod"}
	cmdsNeedGH           = map[string]bool{
		commands.CommandNameFork: true,
		commands.CommandNameDev:  true,
		commands.CommandNamePR:   true,
	}
	cmdsSkipPrerequisites = map[string]bool{
		commands.CommandNameVersion: true,
		commands.CommandNameUpgrade: true,
		"help":                      true,
	}
)
