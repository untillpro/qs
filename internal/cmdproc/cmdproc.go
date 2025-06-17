package cmdproc

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/gitcmds"
	"github.com/untillpro/qs/internal/commands"
	"github.com/untillpro/qs/vcs"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
)

func updateCmd(ctx context.Context, params *qsGlobalParams) *cobra.Command {
	var cfgUpload vcs.CfgUpload
	var uploadCmd = &cobra.Command{
		Use:   commands.CommandNameU,
		Short: pushParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := getWorkingDir(params)
			if err != nil {
				return err
			}

			return commands.U(cmd, cfgUpload, wd)
		},
	}
	uploadCmd.Flags().StringSliceVarP(&cfgUpload.Message, pushMessageWord, pushMessageParam, []string{gitcmds.PushDefaultMsg}, pushMsgComment)

	return uploadCmd
}

func downloadCmd(ctx context.Context, params *qsGlobalParams) *cobra.Command {
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

func releaseCmd(ctx context.Context, params *qsGlobalParams) *cobra.Command {
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

func guiCmd(ctx context.Context, params *qsGlobalParams) *cobra.Command {
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

func prCmd(ctx context.Context, params *qsGlobalParams) *cobra.Command {
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

func versionCmd(ctx context.Context) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameVersion,
		Short: versionParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Version()
		},
	}

	return cmd
}

func upgradeCmd(ctx context.Context) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameUpgrade,
		Short: upgradeParamDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Upgrade()
		},
	}

	return cmd
}

func devCmd(ctx context.Context, params *qsGlobalParams) *cobra.Command {
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

func forkCmd(ctx context.Context, params *qsGlobalParams) *cobra.Command {
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

// ExecRootCmd executes the root command with the given arguments.
// Returns:
// - context.Context: The context of the executed command
// - error: Any error that occurred during execution.
func ExecRootCmd(ctx context.Context, args []string) (context.Context, error) {
	params := &qsGlobalParams{}
	rootCmd := PrepareRootCmd(
		ctx,
		"qs",
		"Quick git wrapper",
		args,
		"",
		updateCmd(ctx, params),
		downloadCmd(ctx, params),
		releaseCmd(ctx, params),
		guiCmd(ctx, params),
		forkCmd(ctx, params),
		devCmd(ctx, params),
		prCmd(ctx, params),
		upgradeCmd(ctx),
		versionCmd(ctx),
	)
	initChangeDirFlags(rootCmd.Commands(), params)

	return ExecCommandAndCatchInterrupt(rootCmd)
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

// ExecCommandAndCatchInterrupt executes the given command and catches interrupts.
// Returns:
// - context.Context: The context of the executed command
// - error: Any error that occurred during execution.
func ExecCommandAndCatchInterrupt(cmd *cobra.Command) (context.Context, error) {
	cmdExec := func(ctx context.Context) (*cobra.Command, error) {
		return cmd.ExecuteContextC(ctx)
	}

	return goAndCatchInterrupt(cmd, cmdExec)

}

// goAndCatchInterrupt runs the given function in a separate goroutine and catches interrupts.
// Returns:
// - context.Context: The context of the executed command
// - error: Any error that occurred during execution.
func goAndCatchInterrupt(cmd *cobra.Command, f func(ctx context.Context) (*cobra.Command, error)) (context.Context, error) {
	var cmdExecuted *cobra.Command

	var signals = make(chan os.Signal, 1)

	ctxWithCancel, cancel := context.WithCancel(cmd.Context())
	signal.Notify(signals, os.Interrupt)

	var err error
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		cmdExecuted, err = f(ctxWithCancel)
		cancel()
	}()

	select {
	case sig := <-signals:
		logger.Info("signal received:", sig)
		cancel()
	case <-ctxWithCancel.Done():
	}
	logger.Verbose("waiting for function to finish...")
	wg.Wait()

	return cmdExecuted.Context(), err
}

func PrepareRootCmd(ctx context.Context, use string, short string, args []string, version string, cmds ...*cobra.Command) *cobra.Command {

	var rootCmd = &cobra.Command{
		Use:   use,
		Short: short,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if ok, _ := cmd.Flags().GetBool("trace"); ok {
				logger.SetLogLevel(logger.LogLevelTrace)
				logger.Verbose("Using logger.LogLevelTrace...")
			} else if ok, _ := cmd.Flags().GetBool("verbose"); ok {
				logger.SetLogLevel(logger.LogLevelVerbose)
				logger.Verbose("Using logger.LogLevelVerbose...")
			}
		},
	}

	var versionCmd = &cobra.Command{
		Use:     "version",
		Short:   "Print current version",
		Aliases: []string{"ver"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}

	rootCmd.SetContext(ctx)
	rootCmd.SetArgs(args[1:])
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(cmds...)
	// Set context for all subcommands
	for _, cmd := range cmds {
		cmd.SetContext(ctx)
	}

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().Bool("trace", false, "Extremely verbose output")
	rootCmd.SilenceUsage = true
	return rootCmd
}
