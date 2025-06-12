package cmdproc

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/cobrau"
	"github.com/untillpro/qs/gitcmds"
	"github.com/untillpro/qs/internal/commands"
	"github.com/untillpro/qs/vcs"
	"os"
	"os/exec"
	"runtime"
)

func updateCmd(params *qsGlobalParams) *cobra.Command {
	var cfgUpload vcs.CfgUpload
	var uploadCmd = &cobra.Command{
		Use:   commands.CommandNameU,
		Short: pushParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := getWorkingDir(params)
			if err != nil {
				return err
			}

			return commands.U(cfgUpload, wd)
		},
	}
	uploadCmd.Flags().StringSliceVarP(&cfgUpload.Message, pushMessageWord, pushMessageParam, []string{gitcmds.PushDefaultMsg}, pushMsgComment)

	return uploadCmd
}

func downloadCmd(params *qsGlobalParams) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameD,
		Short: pullParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := getWorkingDir(params)
			if err != nil {
				return err
			}

			return commands.D(wd)
		},
	}

	return cmd
}

func releaseCmd(params *qsGlobalParams) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameR,
		Short: releaseParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := getWorkingDir(params)
			if err != nil {
				return err
			}

			return commands.R(wd)
		},
	}

	return cmd
}

func guiCmd(params *qsGlobalParams) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameG,
		Short: guiParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := getWorkingDir(params)
			if err != nil {
				return err
			}

			return commands.G(wd)
		},
	}

	return cmd
}

func prCmd(params *qsGlobalParams) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNamePR,
		Short: prParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := getWorkingDir(params)
			if err != nil {
				return err
			}

			return commands.Pr(cmd, wd)
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

func devCmd(params *qsGlobalParams) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameDev,
		Short: devParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := getWorkingDir(params)
			if err != nil {
				return err
			}

			return commands.Dev(cmd, wd, args)
		},
	}
	cmd.Flags().BoolP(devDelParamFull, devDelParam, false, devDelMsgComment)
	cmd.Flags().BoolP(ignorehookDelParamFull, ignorehookDelParam, false, devIgnoreHookMsgComment)
	cmd.Flags().BoolP(noForkParamFull, noForkParam, false, devNoForkMsgComment)

	return cmd
}

func forkCmd(params *qsGlobalParams) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameFork,
		Short: forkParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := getWorkingDir(params)
			if err != nil {
				return err
			}

			return commands.Fork(wd)
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
	params := &qsGlobalParams{}
	rootCmd := cobrau.PrepareRootCmd(
		"qs",
		"Quick git wrapper",
		args,
		"",
		updateCmd(params),
		downloadCmd(params),
		releaseCmd(params),
		guiCmd(params),
		forkCmd(params),
		devCmd(params),
		prCmd(params),
		upgradeCmd(),
		versionCmd(),
	)
	//rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
	//	return prepareParams(cmd, params, args)
	//}
	initChangeDirFlags(rootCmd.Commands(), params)

	return cobrau.ExecCommandAndCatchInterrupt(rootCmd)
}

func getWorkingDir(params *qsGlobalParams) (string, error) {
	if params.Dir != "" {
		return params.Dir, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	return wd, nil
}

func initChangeDirFlags(cmds []*cobra.Command, params *qsGlobalParams) {
	for _, cmd := range cmds {
		if cmd.Name() == "version" {
			continue
		}
		cmd.Flags().StringVarP(&params.Dir, "change-dir", "C", "", "change to dir before running the command. Any files named on the command line are interpreted after changing directories")
	}
}

//func prepareParams(cmd *cobra.Command, params *qsGlobalParams, args []string) (err error) {
//	wd, err := getWorkingDir(params)
//
//	if len(args) > 0 {
//		switch {
//		case strings.Contains(cmd.Use, "init"):
//			params.ModulePath = args[0]
//		case strings.Contains(cmd.Use, "baseline") || strings.Contains(cmd.Use, "compat"):
//			params.TargetDir = filepath.Clean(args[0])
//		}
//	}
//	params.Dir, err = makeAbsPath(params.Dir)
//	if err != nil {
//		return
//	}
//	if params.IgnoreFile != "" {
//		params.IgnoreFile = filepath.Clean(params.IgnoreFile)
//	}
//	if params.TargetDir == "" {
//		params.TargetDir = params.Dir
//	}
//	return nil
//}
