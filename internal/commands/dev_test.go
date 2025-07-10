package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBranchName(t *testing.T) {
	str, _, err := getBranchName(false, "Show", "must", "go", "on", "https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-must-go-on", str)

	str, _, err = getBranchName(false, "Show   ivv?", "must    ", "go", "on---", "https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-ivv-must-go-on", str)

	str, _, err = getBranchName(false, "Show", "must", "go", "on")
	require.NoError(t, err)
	require.Equal(t, "Show-must-go-on", str)

	str, _, err = getBranchName(false, "Show")
	require.NoError(t, err)
	require.Equal(t, "Show", str)

	str, _, err = getBranchName(false, "Show   ivv? must $   go on---  https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-ivv-must-go-on", str)

	str, _, err = getBranchName(false, "Show   ivv? must $   ", "go on---  https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-ivv-must-go-on", str)

	str, _, err = getBranchName(false, "Show   ivv? must $   go  on---", "https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-ivv-must-go-on", str)

	str, _, err = getBranchName(false, "Show", "ivv? must $   go  on--- https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-ivv-must-go-on", str)

	str, _, err = getBranchName(false, "Show", "ivv? must $   go  on---", "https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-ivv-must-go-on", str)

	str, _, err = getBranchName(false, "q", "dev", "https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-q-dev", str)

	str, _, err = getBranchName(false, "q", "dev", "https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-q-dev", str)

	str, _, err = getBranchName(false, "qs: add Kaiser task link to generated commit message", "https://dev.heeus.io/launchpad/#!25947")
	require.NoError(t, err)
	require.Equal(t, "25947-qs-add-Kaiser-task-link-to-generated-commit", str)

	//Logn name
	str, _, err = getBranchName(false, "Show", "me this  very long string more than fifty symbols in lenth with long task number 11111111111111", "https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-me-this-very-long-string-more-than-fift", str)

	//URL name
	str, _, err = getBranchName(false, "https://www.projectkaiser.com/online/#!3206802")
	require.NoError(t, err)
	require.Equal(t, "www-projectkaiser-com-online-#-3206802", str)

	str, _, err = getBranchName(false, "https://github.com/voedger/voedger/issues/395")
	require.NoError(t, err)
	require.Equal(t, "github-com-voedger-voedger-issues-395", str)
}

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
			result, ok := GetJiraTicketIDFromArgs(tt.args...)
			if result != tt.expected || ok != tt.ok {
				t.Errorf("GetJiraTicketIDFromArgs(%v) = (%v, %v), want (%v, %v)", tt.args, result, ok, tt.expected, tt.ok)
			}
		})
	}
}
