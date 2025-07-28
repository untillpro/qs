package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBranchName(t *testing.T) {
	str, _, err := GetBranchName(false, "Show", "must", "go", "on", "https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-must-go-on", str)

	str, _, err = GetBranchName(false, "Show   ivv?", "must    ", "go", "on---", "https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-ivv-must-go-on", str)

	str, _, err = GetBranchName(false, "Show", "must", "go", "on")
	require.NoError(t, err)
	require.Equal(t, "Show-must-go-on", str)

	str, _, err = GetBranchName(false, "Show")
	require.NoError(t, err)
	require.Equal(t, "Show", str)

	str, _, err = GetBranchName(false, "Show   ivv? must $   go on---  https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-ivv-must-go-on", str)

	str, _, err = GetBranchName(false, "Show   ivv? must $   ", "go on---  https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-ivv-must-go-on", str)

	str, _, err = GetBranchName(false, "Show   ivv? must $   go  on---", "https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-ivv-must-go-on", str)

	str, _, err = GetBranchName(false, "Show", "ivv? must $   go  on--- https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-ivv-must-go-on", str)

	str, _, err = GetBranchName(false, "Show", "ivv? must $   go  on---", "https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-ivv-must-go-on", str)

	str, _, err = GetBranchName(false, "q", "dev", "https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-q-dev", str)

	str, _, err = GetBranchName(false, "q", "dev", "https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-q-dev", str)

	str, _, err = GetBranchName(false, "qs: add Kaiser task link to generated commit message", "https://dev.heeus.io/launchpad/#!25947")
	require.NoError(t, err)
	require.Equal(t, "25947-qs-add-Kaiser-task-link-to-generated-commit", str)

	//Logn name
	str, _, err = GetBranchName(false, "Show", "me this  very long string more than fifty symbols in lenth with long task number 11111111111111", "https://dev.heeus.io/launchpad/#!13427")
	require.NoError(t, err)
	require.Equal(t, "13427-Show-me-this-very-long-string-more-than-fift", str)

	//URL name
	str, _, err = GetBranchName(false, "https://www.projectkaiser.com/online/#!3206802")
	require.NoError(t, err)
	require.Equal(t, "www-projectkaiser-com-online-#-3206802", str)

	str, _, err = GetBranchName(false, "https://github.com/voedger/voedger/issues/395")
	require.NoError(t, err)
	require.Equal(t, "github-com-voedger-voedger-issues-395", str)
}

func TestGeRepoNameFromURL(t *testing.T) {
	topicid := GetTaskIDFromURL("https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, "13427", topicid)
}
