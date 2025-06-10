package cmdproc

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/cobrau"
	"github.com/untillpro/qs/git"
	"github.com/untillpro/qs/internal/commands"
	"github.com/untillpro/qs/vcs"
)

func updateCmd() *cobra.Command {
	var cfgUpload vcs.CfgUpload
	var uploadCmd = &cobra.Command{
		Use:   commands.CommandNameU,
		Short: pushParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.U(cfgUpload)
		},
	}
	uploadCmd.Flags().StringSliceVarP(&cfgUpload.Message, pushMessageWord, pushMessageParam, []string{git.PushDefaultMsg}, pushMsgComment)

	return uploadCmd
}

func downloadCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameD,
		Short: pullParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.D()
		},
	}

	return cmd
}

func releaseCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameR,
		Short: releaseParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.R()
		},
	}

	return cmd
}

func guiCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameG,
		Short: guiParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.G()
		},
	}

	return cmd
}

func prCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNamePR,
		Short: prParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Pr(cmd)
		},
	}
	cmd.Flags().BoolP(prdraftParamFull, prdraftParam, false, prdraftMsgComment)

	return cmd
}

func versionCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameVersion,
		Short: versionParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Version()
		},
	}

	return cmd
}

func upgradeCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameUpgrade,
		Short: upgradeParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Upgrade()
		},
	}

	return cmd
}

func devCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameDev,
		Short: devParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Dev(cmd, args)
		},
	}
	cmd.Flags().BoolP(devDelParamFull, devDelParam, false, devDelMsgComment)
	cmd.Flags().BoolP(ignorehookDelParamFull, ignorehookDelParam, false, devIgnoreHookMsgComment)
	cmd.Flags().BoolP(noForkParamFull, noForkParam, false, devNoForkMsgComment)

	return cmd
}

func forkCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameFork,
		Short: forkParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Fork()
		},
	}

	return cmd
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

// CheckCommands verifies if the required commands are installed on the system
func CheckCommands(commands []string) error {
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

func ExecRootCmd(args []string) error {
	rootCmd := cobrau.PrepareRootCmd(
		"qs",
		"Quick git wrapper",
		args,
		"",
		updateCmd(),
		downloadCmd(),
		releaseCmd(),
		guiCmd(),
		forkCmd(),
		devCmd(),
		prCmd(),
		upgradeCmd(),
		versionCmd(),
	)

	return cobrau.ExecCommandAndCatchInterrupt(rootCmd)
}
