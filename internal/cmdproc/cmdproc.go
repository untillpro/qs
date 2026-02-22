package cmdproc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/gitcmds"
	"github.com/untillpro/qs/internal/commands"
	"github.com/untillpro/qs/utils"
)

func updateCmd(_ context.Context, params *qsGlobalParams) *cobra.Command {
	commintMessage := ""
	var uploadCmd = &cobra.Command{
		Use:   commands.CommandNameU,
		Short: "Upload sources to repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := gitcmds.Status(params.Dir); err != nil {
				return err
			}
			return commands.U(cmd, commintMessage, params.Dir)
		},
	}
	uploadCmd.Flags().StringVarP(&commintMessage, "message", "m", "", "Use the given string as the commit message")

	return uploadCmd
}

func downloadCmd(_ context.Context, params *qsGlobalParams) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameD,
		Short: "Download sources from repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			return gitcmds.Download(params.Dir)
		},
	}

	return cmd
}

func releaseCmd(_ context.Context, params *qsGlobalParams) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameR,
		Short: "Create a release",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Release(params.Dir)
		},
	}

	return cmd
}

func guiCmd(_ context.Context, params *qsGlobalParams) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameG,
		Short: "Show GUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return gitcmds.Gui(params.Dir)
		},
	}

	return cmd
}

func prCmd(_ context.Context, params *qsGlobalParams) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNamePR,
		Short: "Make pull request",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ask for confirmation before creating the PR
			var needDraft = false
			if cmd.Flag("draft").Value.String() == "true" {
				needDraft = true
			}

			return gitcmds.Pr(params.Dir, needDraft)
		},
	}
	cmd.Flags().BoolP("draft", "d", false, "Create draft of pull request")

	return cmd
}

func versionCmd(_ context.Context) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameVersion,
		Short: "Print qs version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Version()
		},
	}

	return cmd
}

func upgradeCmd(_ context.Context) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameUpgrade,
		Short: "Print command to upgrade qs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Upgrade()
		},
	}

	return cmd
}

func devCmd(_ context.Context, params *qsGlobalParams) *cobra.Command {
	doDelete := false
	ignoreHook := false
	var cmd = &cobra.Command{
		Use:   commands.CommandNameDev,
		Short: "Create developer branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Dev(cmd, params.Dir, doDelete, ignoreHook, args)
		},
	}

	cmd.Flags().BoolVarP(&doDelete, "delete", "d", false, "Deletes all merged branches from forked repository")
	cmd.Flags().BoolVarP(&ignoreHook, "ignore-hook", "i", false, "Ignore creating local hook")

	return cmd
}

func forkCmd(_ context.Context, params *qsGlobalParams) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   commands.CommandNameFork,
		Short: "Fork original repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Fork(params.Dir)
		},
	}

	return cmd
}

// checkRequiredBashCommands checks if all required bash commands are available
func checkRequiredBashCommands() error {
	missing := []string{}
	for _, cmd := range requiredBashCommands {
		_, err := exec.LookPath(cmd)
		if err != nil {
			missing = append(missing, cmd)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing following commands: %s\nSee https://github.com/untillpro/qs?tab=readme-ov-file#git", missing)
	}

	return nil
}

// ExecRootCmd executes the root command with the given arguments.
// Returns:
// - context.Context: The context of the executed command
// - error: Any error that occurred during execution.
func ExecRootCmd(ctx context.Context, args []string) (context.Context, error) {
	params := &qsGlobalParams{}
	rootCmd, err := PrepareRootCmd(
		ctx,
		"qs",
		"Quick git wrapper",
		args,
		"",
		params,
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
	if err != nil {
		return nil, err
	}

	return ExecCommandAndCatchInterrupt(rootCmd)
}

func initChangeDirFlags(cmds []*cobra.Command, params *qsGlobalParams) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	for _, cmd := range cmds {
		if cmd.Name() == "version" {
			continue
		}
		cmd.Flags().StringVarP(&params.Dir, "change-dir", "C", wd, "change to dir before running the command. Any files named on the command line are interpreted after changing directories")
	}
	return nil
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
	wg.Wait()

	return cmdExecuted.Context(), err
}

func PrepareRootCmd(ctx context.Context, use string, short string, args []string, version string, params *qsGlobalParams, cmds ...*cobra.Command) (*cobra.Command, error) {
	var rootCmd = &cobra.Command{
		Use:   use,
		Short: short,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Set log level first - handle all log level options
			if ok, _ := cmd.Flags().GetBool("trace"); ok {
				logger.SetLogLevel(logger.LogLevelTrace)
				logger.Verbose("Using logger.LogLevelTrace...")
			} else if ok, _ := cmd.Flags().GetBool("verbose"); ok {
				logger.SetLogLevel(logger.LogLevelVerbose)
				logger.Verbose("Using logger.LogLevelVerbose...")
			} else {
				// Default log level
				logger.SetLogLevel(logger.LogLevelInfo)
			}

			// Skip checks for commands that don't need them
			if cmdsSkipPrerequisites[cmd.Name()] {
				return nil
			}

			// Check required bash commands
			if err := checkRequiredBashCommands(); err != nil {
				return err
			}

			// Check QS version (unless skipped)
			skipQsVerCheck, _ := strconv.ParseBool(os.Getenv(commands.EnvSkipQsVersionCheck))
			if !skipQsVerCheck && !commands.CheckQsVer() {
				fmt.Println("Ok, see you")
				os.Exit(1)
			}

			// Check GitHub CLI (for commands that need it)
			if cmdsNeedGH[cmd.Name()] {
				if err := utils.CheckGH(); err != nil {
					return err
				}
			}

			if cmd.Name() != commands.CommandNameUpgrade && cmd.Name() != commands.CommandNameVersion {
				ok, err := gitcmds.CheckIfGitRepo(params.Dir)
				if err != nil {
					return err
				}
				if !ok {
					return errors.New("this is not a git repository")
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return gitcmds.Status(params.Dir)
		},
	}

	rootCmd.SetContext(ctx)
	rootCmd.SetArgs(args[1:])
	rootCmd.AddCommand(cmds...)
	// Set context for all subcommands
	for _, cmd := range cmds {
		cmd.SetContext(ctx)
	}

	rootCmd.PersistentFlags().BoolVarP(&commands.Verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().Bool("trace", false, "Extremely verbose output")
	rootCmd.SilenceUsage = true
	err := initChangeDirFlags(rootCmd.Commands(), params)
	return rootCmd, err
}
