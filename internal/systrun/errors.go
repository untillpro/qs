package systrun

import "errors"

var (
	ErrEmptyInitialRepoConfig = errors.New("initialRepoConfig is empty")
	ErrEmptyGithubAccount     = errors.New("Github account is empty")
	ErrEmptyGithubToken       = errors.New("Github token is empty")
	ErrEmptyRepoName          = errors.New("Repo name is empty")
)
