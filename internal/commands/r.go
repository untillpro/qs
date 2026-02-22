/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package commands

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/utils"
)

/*
	- Pull
	- Get current verson
	- If PreRelease is not empty fails
	- Calculate target version
	- Ask
	- Save version
	- Commit
	- Tag with target version
	- Bump current version
	- Commit
	- Push commits and tags
*/

// Release current branch. Remove PreRelease, tag, bump version, push
func Release(wd string) error {

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Pulling")
	stdout, stderr, err := new(exec.PipedExec).
		Command("git", "pull").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("error pulling: %w", err)
	}
	logger.Verbose(stdout)

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Reading current version")
	currentVersion, err := utils.ReadVersion()
	if err != nil {
		return fmt.Errorf("error reading file 'version': %w", err)
	}
	if len(currentVersion.PreRelease) == 0 {
		return errors.New("pre-release part of version does not exist: " + currentVersion.String())
	}

	// Calculate target version

	targetVersion := currentVersion
	targetVersion.PreRelease = ""

	fmt.Printf("Version %v will be tagged, bumped and pushed, agree? [y]", targetVersion)
	var response string
	_, _ = fmt.Scanln(&response)
	if response != "y" {
		return errors.New("release aborted by user")
	}

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Updating 'version' file")
	if err := targetVersion.Save(); err != nil {
		return fmt.Errorf("error saving file 'version': %w", err)
	}

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Committing target version")
	{
		params := []string{"commit", "-a", "-m", "#scm-ver " + targetVersion.String()}
		stdout, stderr, err = new(exec.PipedExec).
			Command("git", params...).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("error committing target version: %w", err)
		}
		logger.Verbose(stdout)
	}

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Tagging")
	{
		tagName := "v" + targetVersion.String()
		n := time.Now()
		params := []string{"tag", "-m", "Version " + tagName + " of " + n.Format("2006/01/02 15:04:05"), tagName}
		stdout, stderr, err = new(exec.PipedExec).
			Command("git", params...).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("error tagging version: %w", err)
		}
		logger.Verbose(stdout)
	}

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Bumping version")
	newVersion := currentVersion
	{
		newVersion.Minor++
		newVersion.PreRelease = "SNAPSHOT"
		if err := newVersion.Save(); err != nil {
			return fmt.Errorf("error saving file 'version': %w", err)
		}
	}

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Committing new version")
	{
		params := []string{"commit", "-a", "-m", "#scm-ver " + newVersion.String()}
		stdout, stderr, err = new(exec.PipedExec).
			Command("git", params...).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("error committing new version: %w", err)
		}
	}

	// *************************************************
	_, _ = fmt.Fprintln(os.Stdout, "Pushing to origin")
	{
		params := []string{"push", "--follow-tags", "origin"}
		err = utils.Retry(func() error {
			stdout, stderr, err = new(exec.PipedExec).
				Command("git", params...).
				WorkingDir(wd).
				RunToStrings()
			if err != nil {
				if len(stderr) > 0 {
					return errors.New(stderr)
				}

				return fmt.Errorf("error pushing to origin: %w", err)
			}

			return nil
		})
		if err != nil {
			if len(stderr) > 0 {
				logger.Verbose(stderr)
			}

			return err
		}
		logger.Verbose(stdout)
	}

	return nil
}
