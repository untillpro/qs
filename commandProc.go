package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	cobra "github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/git"
	"github.com/untillpro/qs/internal/commands"
	"github.com/untillpro/qs/vcs"
)

const (
	utilityName = "qs"                //root command name
	utilityDesc = "Quick git wrapper" //root command description

	pushParamDesc    = "Upload sources to repo"
	pushMessageWord  = "message"
	pushMessageParam = "m"
	pushMsgComment   = `Use the given string as the commit message. If multiple -m options are given
 their values are concatenated as separate paragraphs`

	pullParamDesc = "Download sources from repo"

	releaseParamDesc = "Create a release"

	guiParamDesc = "Show GUI"

	verboseWord  = "verbose"
	verboseParam = "v"
	verboseDesc  = "verbose output"

	forkParamDesc = "Fork original repo"

	devDelParam            = "d"
	devDelParamFull        = "delete"
	ignorehookDelParam     = "i"
	ignorehookDelParamFull = "ignore-hook"
	prdraftParam           = "d"
	prdraftParamFull       = "draft"
	noForkParam            = "n"
	noForkParamFull        = "no-fork"

	prParamDesc = "Make pull request"

	devDelMsgComment        = "Deletes all merged branches from forked repository"
	devIgnoreHookMsgComment = "Ignore creating local hook"
	devNoForkMsgComment     = "Allows to create branch in main repo"
	prdraftMsgComment       = "Create draft of pull request"
	devParamDesc            = "Create developer branch"
	upgradeParamDesc        = "Print command to upgrade qs"
	versionParamDesc        = "Print qs version"
)

var verbose bool

func globalConfig() {
	if verbose {
		logger.SetLogLevel(logger.LogLevelVerbose)
	} else {
		logger.SetLogLevel(logger.LogLevelInfo)
	}
}

type commandProcessor struct {
	cfgStatus vcs.CfgStatus
	rootcmd   *cobra.Command
}

// BuildCommandProcessor s.e.
func buildCommandProcessor() *commandProcessor {
	cp := commandProcessor{}
	return cp.setRootCmd()
}

func (cp *commandProcessor) setRootCmd() *commandProcessor {
	cp.rootcmd = &cobra.Command{
		Use:   utilityName,
		Short: utilityDesc,
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig()
			git.Status(cp.cfgStatus)
		},
	}
	cp.rootcmd.PersistentFlags().BoolVarP(&verbose, verboseWord, verboseParam, false, verboseDesc)
	return cp
}

func (cp *commandProcessor) addUpdateCmd() *commandProcessor {
	var cfgUpload vcs.CfgUpload
	var uploadCmd = &cobra.Command{
		Use:   commands.CommandNameU,
		Short: pushParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			commands.U(cp.cfgStatus, cfgUpload, args)
		},
	}

	uploadCmd.Flags().StringSliceVarP(&cfgUpload.Message, pushMessageWord, pushMessageParam, []string{git.PushDefaultMsg}, pushMsgComment)
	cp.rootcmd.AddCommand(uploadCmd)
	return cp
}

func (cp *commandProcessor) addDownloadCmd() *commandProcessor {
	var cfg vcs.CfgDownload
	var cmd = &cobra.Command{
		Use:   commands.CommandNameD,
		Short: pullParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			commands.D(cfg)
		},
	}
	cp.rootcmd.AddCommand(cmd)
	return cp
}

func (cp *commandProcessor) addReleaseCmd() *commandProcessor {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameR,
		Short: releaseParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			commands.R()
		},
	}
	cp.rootcmd.AddCommand(cmd)
	return cp
}

func (cp *commandProcessor) addGUICmd() *commandProcessor {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameG,
		Short: guiParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			commands.G()
		},
	}
	cp.rootcmd.AddCommand(cmd)
	return cp
}

func (cp *commandProcessor) Execute() {
	if cp.rootcmd == nil {
		return
	}
	if len(cp.rootcmd.Commands()) == 0 {
		return
	}
	err := cp.rootcmd.Execute()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func (cp *commandProcessor) addPr() *commandProcessor {
	var cmd = &cobra.Command{
		Use:   commands.CommandNamePR,
		Short: prParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			commands.Pr(cmd, args)
		},
	}
	cmd.Flags().BoolP(prdraftParamFull, prdraftParam, false, prdraftMsgComment)
	cp.rootcmd.AddCommand(cmd)
	return cp
}

func (cp *commandProcessor) addVersion() *commandProcessor {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameVersion,
		Short: versionParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			commands.Version()
		},
	}
	cp.rootcmd.AddCommand(cmd)
	return cp
}

func (cp *commandProcessor) addUpgrade() *commandProcessor {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameUpgrade,
		Short: upgradeParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			commands.Upgrade()
		},
	}
	cp.rootcmd.AddCommand(cmd)
	return cp
}

func (cp *commandProcessor) addDevBranch() *commandProcessor {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameDev,
		Short: devParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			commands.Dev(cmd, args)
		},
	}
	cmd.Flags().BoolP(devDelParamFull, devDelParam, false, devDelMsgComment)
	cp.rootcmd.AddCommand(cmd)
	cmd.Flags().BoolP(ignorehookDelParamFull, ignorehookDelParam, false, devIgnoreHookMsgComment)
	cmd.Flags().BoolP(noForkParamFull, noForkParam, false, devNoForkMsgComment)

	return cp
}

func (cp *commandProcessor) addForkBranch() *commandProcessor {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameFork,
		Short: forkParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			commands.Fork()
		},
	}
	cp.rootcmd.AddCommand(cmd)
	return cp
}

// redText returns the given text wrapped in ANSI escape codes (for Linux/macOS)
// or formatted for Windows.
func redText(text string) string {
	if runtime.GOOS == "windows" {
		// Windows: Use cmd ANSI sequences if supported, otherwise just return text
		return "\033[31m" + text + "\033[0m"
	}
	// Linux/macOS ANSI escape codes for red text
	return "\033[31m" + text + "\033[0m"
}

// checkCommands verifies if the required commands are installed on the system
func checkCommands(commands []string) error {
	missing := []string{}
	for _, cmd := range commands {
		_, err := exec.LookPath(cmd)
		if err != nil {
			missing = append(missing, cmd)
		}
	}

	if len(missing) > 0 {
		if len(missing) == 1 {
			return fmt.Errorf(redText("Error: missing required command: %s"), missing[0])
		} else {
			return fmt.Errorf(redText("Error: missing required commands: %v"), missing)
		}
	}
	return nil
}
