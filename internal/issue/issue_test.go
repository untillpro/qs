package issue

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseIssueFromArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantType IssueType
		wantID   string
		wantNum  int
		wantErr  bool
	}{
		{
			name:     "Multiple args returns FreeForm",
			args:     []string{"arg1", "arg2"},
			wantType: FreeForm,
		},
		{
			name:     "Plain text returns FreeForm",
			args:     []string{"some-feature-branch"},
			wantType: FreeForm,
		},
		{
			name:     "Jira URL returns Jira type",
			args:     []string{"https://untill.atlassian.net/browse/AIR-270"},
			wantType: Jira,
			wantID:   "AIR-270",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			info, err := ParseIssueFromArgs(".", tt.args...)
			if tt.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)
			require.Equal(tt.wantType, info.Type)
			if tt.wantID != "" {
				require.Equal(tt.wantID, info.ID)
			}
			if tt.wantNum != 0 {
				require.Equal(tt.wantNum, info.Number)
			}
		})
	}
}
