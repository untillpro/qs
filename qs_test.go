package main

import (
	"strings"
	"testing"

	"github.com/atotto/clipboard"
	"github.com/stretchr/testify/assert"
	"github.com/untillpro/qs/git"
)

func TestClipBoard(t *testing.T) {
	err := clipboard.WriteAll("1,2,3,5")
	assert.Empty(t, err)
	arg, _ := clipboard.ReadAll()

	args := strings.Split(arg, "\n")
	var newarg string
	for _, str := range args {
		newarg += str
		newarg += " "
	}
	assert.NotEmpty(t, newarg)
}

func TestIssueRepoFromURL(t *testing.T) {
	repo := git.GetIssuerepoFromUrl("https://github.com/untillpro/qs/issues/24")
	assert.Equal(t, "untillpro/qs", repo)
}
