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
		wantErr  bool
	}{
		{
			name:     "Multiple args returns FreeForm",
			args:     []string{"arg1"},
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
			info, err := ParseIssueFromArgs(tt.args...)
			if tt.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)
			require.Equal(tt.wantType, info.Type)
			if tt.wantID != "" {
				require.Equal(tt.wantID, info.ID)
			}
		})
	}
}

func TestBuildDevBranchName(t *testing.T) {
	tests := []struct {
		name       string
		info       IssueInfo
		wantBranch string
	}{
		{
			name:       "Single word",
			info:       IssueInfo{Type: FreeForm, Text: "Show"},
			wantBranch: "show-dev",
		},
		{
			name:       "Multiple words",
			info:       IssueInfo{Type: FreeForm, Text: "Show must go on"},
			wantBranch: "show-must-go-on-dev",
		},
		{
			name:       "Words with special chars",
			info:       IssueInfo{Type: FreeForm, Text: "Show   ivv? must    go on---"},
			wantBranch: "show-ivv-must-go-on-dev",
		},
		{
			name:       "Text with PK URL",
			info:       IssueInfo{Type: FreeForm, Text: "Show must go on https://dev.heeus.io/launchpad/#!13427"},
			wantBranch: "show-must-go-on-https-dev-heeus-io-launchpad-13427-dev",
		},
		{
			name:       "Text with special chars and PK URL",
			info:       IssueInfo{Type: FreeForm, Text: "Show   ivv? must $   go on---  https://dev.heeus.io/launchpad/#!13427"},
			wantBranch: "show-ivv-must-go-on-https-dev-heeus-io-launchpad-1-dev",
		},
		{
			name:       "Long text with PK URL",
			info:       IssueInfo{Type: FreeForm, Text: "Show me this  very long string more than fifty symbols in lenth with long task number 11111111111111 https://dev.heeus.io/launchpad/#!13427"},
			wantBranch: "show-me-this-very-long-string-more-than-fifty-symb-dev",
		},
		{
			name:       "PK URL only",
			info:       IssueInfo{Type: FreeForm, Text: "https://www.projectkaiser.com/online/#!3206802"},
			wantBranch: "https-www-projectkaiser-com-online-3206802-dev",
		},
		{
			name:       "GitHub issues URL as FreeForm",
			info:       IssueInfo{Type: FreeForm, Text: "https://github.com/voedger/voedger/issues/395"},
			wantBranch: "https-github-com-voedger-voedger-issues-395-dev",
		},
		{
			name:       "Short text with PK URL",
			info:       IssueInfo{Type: FreeForm, Text: "q dev https://dev.heeus.io/launchpad/#!13427"},
			wantBranch: "q-dev-https-dev-heeus-io-launchpad-13427-dev",
		},
		{
			name:       "Long description with PK URL",
			info:       IssueInfo{Type: FreeForm, Text: "qs: add Kaiser task link to generated commit message https://dev.heeus.io/launchpad/#!25947"},
			wantBranch: "qs-add-kaiser-task-link-to-generated-commit-messag-dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			branch, _, err := BuildDevBranchName(tt.info)
			require.NoError(err)
			require.Equal(tt.wantBranch, branch)
		})
	}
}
