package commands

import "errors"

var (
	ErrEmptyCommitMessage = errors.New("commit message is missing, use -m to specify")
	ErrShortCommitMessage = errors.New("commit message is missing or too short (minimum 8 characters)")
)
