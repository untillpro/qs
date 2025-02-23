package main

import (
	"os"
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

func TestContainsJiraName(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
		ok       bool
	}{
		{
			name:     "Valid JIRA issue URL",
			args:     []string{"https://untill.atlassian.net/browse/AIR-270"},
			expected: "AIR-270",
			ok:       true,
		},
		{
			name:     "Multiple arguments with valid JIRA issue URL",
			args:     []string{"random-text", "https://voedger.atlassian.net/browse/TRE-FISH-270"},
			expected: "TRE-FISH-270",
			ok:       true,
		},
		{
			name:     "JIRA URL with description",
			args:     []string{"My name of issue https://untill.atlassian.net/browse/AIR-270"},
			expected: "AIR-270",
			ok:       true,
		},
		{
			name:     "No JIRA issue URL",
			args:     []string{"random-text", "another-arg"},
			expected: "",
			ok:       false,
		},
		{
			name:     "Empty arguments",
			args:     []string{},
			expected: "",
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := argContainsJiraIssue(tt.args...)
			if result != tt.expected || ok != tt.ok {
				t.Errorf("argContainsJiraIssue(%v) = (%v, %v), want (%v, %v)", tt.args, result, ok, tt.expected, tt.ok)
			}
		})
	}
}

func TestGetJiraIssueName(t *testing.T) {
	os.Setenv("JIRA_EMAIL", "v.istratenko@dev.untill.com")
	name := getJiraIssueNameByNumber("AIR-270")
	if name == "" {
		assert.Equal(t, "qwfwf", name)
	}
}

// Helper function to compare two slices
func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
