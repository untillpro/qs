/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package systrun

import (
	"errors"
	"fmt"

	goGitPkg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/untillpro/qs/gitcmds"
)

// createRemote creates a remote in the cloned repository
func CreateRemote(wd, remote, account, token, repoName string, isUpstream bool) error {
	repo, err := gitcmds.OpenGitRepository(wd)
	if err != nil {
		return err
	}

	if err = repo.DeleteRemote(remote); err != nil {
		if !errors.Is(err, goGitPkg.ErrRemoteNotFound) {
			return fmt.Errorf("failed to delete %s remote: %w", remote, err)
		}
	}

	remoteURL := BuildRemoteURL(account, token, repoName, isUpstream)
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: remote,
		URLs: []string{remoteURL},
	})
	if err != nil {
		return fmt.Errorf("failed to create %s remote: %w", remote, err)
	}

	return nil
}

// buildRemoteURL constructs the remote URL for cloning
func BuildRemoteURL(account, token, repoName string, isUpstream bool) string {
	return "https://" + account + ":" + token + "@github.com/" + account + slash + repoName + ".git"
}
