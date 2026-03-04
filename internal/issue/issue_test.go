package issue

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractURLFromText(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		wantURL           string
		wantRemainingText string
	}{
		{
			name:              "Text + URL on same line",
			input:             "Show must go on https://dev.untill.com/projects/#!763090",
			wantURL:           "https://dev.untill.com/projects/#!763090",
			wantRemainingText: "Show must go on",
		},
		{
			name:              "Text + URL on separate lines",
			input:             "Show must go on\nhttps://dev.untill.com/projects/#!763090",
			wantURL:           "https://dev.untill.com/projects/#!763090",
			wantRemainingText: "Show must go on",
		},
		{
			name:              "URL only",
			input:             "https://dev.untill.com/projects/#!763090",
			wantURL:           "https://dev.untill.com/projects/#!763090",
			wantRemainingText: "",
		},
		{
			name:              "Text only",
			input:             "Fix authentication bug",
			wantURL:           "",
			wantRemainingText: "Fix authentication bug",
		},
		{
			name:              "Multiple URLs - extracts first",
			input:             "Fix https://first.example.com and https://second.example.com",
			wantURL:           "https://first.example.com",
			wantRemainingText: "Fix  and https://second.example.com",
		},
		{
			name:              "Empty input",
			input:             "",
			wantURL:           "",
			wantRemainingText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			gotURL, gotRemaining := extractURLFromText(tt.input)
			require.Equal(tt.wantURL, gotURL)
			require.Equal(tt.wantRemainingText, gotRemaining)
		})
	}
}

func TestExtractIDFromURL(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		wantID string
	}{
		{
			name:   "ProjectKaiser fragment with #!",
			url:    "https://dev.untill.com/projects/#!763090",
			wantID: "763090",
		},
		{
			name:   "Heeus launchpad fragment",
			url:    "https://dev.heeus.io/launchpad/#!13427",
			wantID: "13427",
		},
		{
			name:   "Jira browse URL",
			url:    "https://org.atlassian.net/browse/AIR-270",
			wantID: "AIR-270",
		},
		{
			name:   "GitHub issue URL",
			url:    "https://github.com/org/repo/issues/42",
			wantID: "42",
		},
		{
			name:   "Empty URL",
			url:    "",
			wantID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantID, ExtractIDFromURL(tt.url))
		})
	}
}

func TestParseIssueFromArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantType IssueType
		wantID   string
		wantURL  string
		wantText string
		wantErr  bool
	}{
		{
			name:     "Fetchable URL only - GitHub",
			args:     []string{"https://github.com/untillpro/qs/issues/123"},
			wantType: GitHub,
			wantID:   "123",
			wantURL:  "https://github.com/untillpro/qs/issues/123",
			wantText: "",
		},
		{
			name:     "Fetchable URL only - Jira",
			args:     []string{"https://untill.atlassian.net/browse/AIR-270"},
			wantType: Jira,
			wantID:   "AIR-270",
			wantURL:  "https://untill.atlassian.net/browse/AIR-270",
			wantText: "",
		},
		{
			name:     "Text only",
			args:     []string{"some-feature-branch"},
			wantType: FreeForm,
			wantURL:  "",
			wantText: "some-feature-branch",
		},
		{
			name:     "Text + GitHub URL",
			args:     []string{"Fix bug https://github.com/untillpro/qs/issues/123"},
			wantType: GitHub,
			wantID:   "123",
			wantURL:  "https://github.com/untillpro/qs/issues/123",
			wantText: "Fix bug",
		},
		{
			name:     "Text + Jira URL",
			args:     []string{"Fix bug https://untill.atlassian.net/browse/AIR-270"},
			wantType: Jira,
			wantID:   "AIR-270",
			wantURL:  "https://untill.atlassian.net/browse/AIR-270",
			wantText: "Fix bug",
		},
		{
			name:     "Text + non-fetchable URL",
			args:     []string{"Show must go on https://dev.untill.com/projects/#!763090"},
			wantType: FreeForm,
			wantID:   "763090",
			wantURL:  "https://dev.untill.com/projects/#!763090",
			wantText: "Show must go on",
		},
		{
			name:    "Non-fetchable URL only - error",
			args:    []string{"https://dev.untill.com/projects/#!763090"},
			wantErr: true,
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
			require.Equal(tt.wantURL, info.URL)
			require.Equal(tt.wantText, info.Text)
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
			name:       "Text with non-fetchable URL - uses Text for branch name",
			info:       IssueInfo{Type: FreeForm, Text: "Show must go on", URL: "https://dev.heeus.io/launchpad/#!13427"},
			wantBranch: "show-must-go-on-dev",
		},
		{
			name:       "Text with special chars and non-fetchable URL",
			info:       IssueInfo{Type: FreeForm, Text: "Show   ivv? must $   go on---", URL: "https://dev.heeus.io/launchpad/#!13427"},
			wantBranch: "show-ivv-must-go-on-dev",
		},
		{
			name:       "Long text with non-fetchable URL - truncated at 50 chars",
			info:       IssueInfo{Type: FreeForm, Text: "Show me this  very long string more than fifty symbols in lenth with long task number 11111111111111", URL: "https://dev.heeus.io/launchpad/#!13427"},
			wantBranch: "show-me-this-very-long-string-more-than-fifty-symb-dev",
		},
		{
			name:       "Short text with non-fetchable URL",
			info:       IssueInfo{Type: FreeForm, Text: "q dev", URL: "https://dev.heeus.io/launchpad/#!13427"},
			wantBranch: "q-dev-dev",
		},
		{
			name:       "Long description with non-fetchable URL - truncated at 50 chars",
			info:       IssueInfo{Type: FreeForm, Text: "qs: add Kaiser task link to generated commit message", URL: "https://dev.heeus.io/launchpad/#!25947"},
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
