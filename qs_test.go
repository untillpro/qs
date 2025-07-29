package main

import (
	"strings"
	"testing"

	"github.com/atotto/clipboard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/untillpro/qs/gitcmds"
)

func TestClipBoard(t *testing.T) {
	err := clipboard.WriteAll("1,2,3,5")
	require.NoError(t, err)
	arg, _ := clipboard.ReadAll()

	args := strings.Split(arg, "\n")
	var newarg string
	for _, str := range args {
		newarg += str
		newarg += " "
	}
	require.NotEmpty(t, newarg)
}

func TestIssueRepoFromURL(t *testing.T) {
	repo := gitcmds.GetIssueRepoFromURL("https://github.com/untillpro/qs/issues/24")
	assert.Equal(t, "untillpro/qs", repo)
}
