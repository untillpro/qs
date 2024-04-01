package main

import (
	"strings"
	"testing"

	"github.com/atotto/clipboard"
	"github.com/stretchr/testify/assert"
	"github.com/untillpro/qs/git"
)

func TestDeleteDup(t *testing.T) {
	str := deleteDupMinus("13427-Show--must----go---on")
	assert.Equal(t, "13427-Show-must-go-on", str)
	str = deleteDupMinus("----Show--must----")
	assert.Equal(t, "-Show-must-", str)
}
func TestGeRepoNameFromURL(t *testing.T) {
	topicid := getTaskIDFromURL("https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, "13427", topicid)
}

func TestGetBranchName(t *testing.T) {
	str, _ := getBranchName(false, "Show", "must", "go", "on", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, "13427-Show-must-go-on", str)
	str, _ = getBranchName(false, "Show   ivv?", "must    ", "go", "on---", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, "13427-Show-ivv-must-go-on", str)
	str, _ = getBranchName(false, "Show", "must", "go", "on")
	assert.Equal(t, "Show-must-go-on", str)
	str, _ = getBranchName(false, "Show")
	assert.Equal(t, "Show", str)
	str, _ = getBranchName(false, "Show   ivv? must $   go on---  https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, "13427-Show-ivv-must-go-on", str)
	str, _ = getBranchName(false, "Show   ivv? must $   ", "go on---  https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, "13427-Show-ivv-must-go-on", str)
	str, _ = getBranchName(false, "Show   ivv? must $   go  on---", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, "13427-Show-ivv-must-go-on", str)
	str, _ = getBranchName(false, "Show", "ivv? must $   go  on--- https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, "13427-Show-ivv-must-go-on", str)
	str, _ = getBranchName(false, "Show", "ivv? must $   go  on---", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, "13427-Show-ivv-must-go-on", str)
	str, _ = getBranchName(false, "q", "dev", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, "13427-q-dev", str)
	str, _ = getBranchName(false, "q", "dev", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, "13427-q-dev", str)
	str, _ = getBranchName(false, "qs: add Kaiser task link to generated commit message", "https://dev.heeus.io/launchpad/#!25947")
	assert.Equal(t, "25947-qs-add-Kaiser-task-link-to-generated-commit", str)

	//Logn name
	str, _ = getBranchName(false, "Show", "me this  very long string more than fifty symbols in lenth with long task number 11111111111111", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, "13427-Show-me-this-very-long-string-more-than-fift", str)

	//URL name
	str, _ = getBranchName(false, "https://www.projectkaiser.com/online/#!3206802")
	assert.Equal(t, "www-projectkaiser-com-online-#-3206802", str)

	str, _ = getBranchName(false, "https://github.com/voedger/voedger/issues/395")
	assert.Equal(t, "github-com-voedger-voedger-issues-395", str)

}

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
