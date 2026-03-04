package notes_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/qs/internal/notes"
)

func TestSerialize(t *testing.T) {
	tests := []struct {
		name        string
		issueURL    string
		description string
		branchType  notes.BranchType
		wantErr     bool
	}{
		{
			name:        "GitHub issue URL",
			issueURL:    "https://github.com/org/repo/issues/1",
			description: "Fix login bug",
			branchType:  notes.BranchTypeDev,
			wantErr:     false,
		},
		{
			name:        "Jira ticket URL",
			issueURL:    "https://org.atlassian.net/browse/AIR-270",
			description: "Live cluster restart",
			branchType:  notes.BranchTypeDev,
			wantErr:     false,
		},
		{
			name:       "Empty URL",
			issueURL:   "",
			branchType: notes.BranchTypePr,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized, err := notes.Serialize(tt.issueURL, tt.branchType, tt.description)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, serialized)

			// Verify the serialized string can be unmarshalled back
			var n notes.Notes
			err = json.Unmarshal([]byte(serialized), &n)
			require.NoError(t, err)

			// Verify the contents
			require.Equal(t, tt.issueURL, n.IssueURL)
			require.Equal(t, tt.description, n.Description)
			require.Equal(t, tt.branchType, n.BranchType)
			require.NotEmpty(t, n.Version)
			// Legacy fields must not be written by new code
			require.Empty(t, n.GithubIssueURL)
			require.Empty(t, n.JiraTicketURL)
		})
	}
}

func TestDeserialize(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected *notes.Notes
		wantErr  bool
	}{
		{
			name:  "Simple JSON in one line",
			input: []string{`{"version":"1.0","github_issue_url":"https://github.com/org/repo/issues/1","jira_ticket_url":"https://jira.org/browse/ISSUE-1","branch_type":0}`},
			expected: &notes.Notes{
				Version:        "1.0",
				GithubIssueURL: "https://github.com/org/repo/issues/1",
				JiraTicketURL:  "https://jira.org/browse/ISSUE-1",
				BranchType:     0,
			},
			wantErr: false,
		},
		{
			name: "JSON split across multiple lines",
			input: []string{
				"Some text before {\"version\":\"1.0\",",
				"\"github_issue_url\":\"https://github.com/org/repo/issues/2\",",
				"\"jira_ticket_url\":\"\",\"branch_type\":1}",
			},
			expected: &notes.Notes{
				Version:        "1.0",
				GithubIssueURL: "https://github.com/org/repo/issues/2",
				JiraTicketURL:  "",
				BranchType:     1,
			},
			wantErr: false,
		},
		{
			name:     "Invalid JSON",
			input:    []string{"Not a valid JSON"},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Empty input",
			input:    []string{},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Unsupported version",
			input:    []string{`{"version":"9.9","github_issue_url":"","jira_ticket_url":"","branch_type":1}`},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := notes.Deserialize(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, tt.expected.Version, got.Version)
			require.Equal(t, tt.expected.GithubIssueURL, got.GithubIssueURL)
			require.Equal(t, tt.expected.JiraTicketURL, got.JiraTicketURL)
			require.Equal(t, tt.expected.BranchType, got.BranchType)
		})
	}
}

func TestReadNotes(t *testing.T) {
	tests := []struct {
		name            string
		input           []string
		wantBranchType  notes.BranchType
		wantDescription string
		wantURL         string
		wantErr         bool
	}{
		{
			name:            "JSON format - pure JSON blob",
			input:           []string{`{"version":"1.0","github_issue_url":"https://github.com/org/repo/issues/1","jira_ticket_url":"","branch_type":1,"description":"My feature"}`},
			wantBranchType:  notes.BranchTypeDev,
			wantDescription: "My feature",
			wantURL:         "https://github.com/org/repo/issues/1",
		},
		{
			name:            "JSON format - issue_url set",
			input:           []string{`{"version":"1.0","github_issue_url":"","jira_ticket_url":"","branch_type":1,"description":"My feature","issue_url":"https://example.com/123"}`},
			wantBranchType:  notes.BranchTypeDev,
			wantDescription: "My feature",
			wantURL:         "https://example.com/123",
		},
		{
			name:            "Old GH issue format - Resolves issue marker",
			input:           []string{"Resolves issue #324 My Best problem ever", "Resolves #324"},
			wantBranchType:  notes.BranchTypeDev,
			wantDescription: "Resolves issue #324 My Best problem ever",
			wantURL:         "",
		},
		{
			name:            "Old PK/custom format - plain text + URL",
			input:           []string{"Permanent support for Peter, Pascal", "https://dev.untill.com/projects/#!361164"},
			wantBranchType:  notes.BranchTypeDev,
			wantDescription: "Permanent support for Peter, Pascal",
			wantURL:         "https://dev.untill.com/projects/#!361164",
		},
		{
			name:    "Empty input",
			input:   []string{},
			wantErr: true,
		},
		{
			name:    "Only empty lines",
			input:   []string{"", "  "},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := notes.ReadNotes(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, tt.wantBranchType, got.BranchType)
			require.Equal(t, tt.wantDescription, got.Description)
			// For JSON format IssueURL is separate from GithubIssueURL; check all URL fields
			gotURL := got.IssueURL
			if gotURL == "" {
				gotURL = got.GithubIssueURL
			}
			if gotURL == "" {
				gotURL = got.JiraTicketURL
			}
			require.Equal(t, tt.wantURL, gotURL)
		})
	}
}

func TestNotesString(t *testing.T) {
	n := notes.Notes{
		Version:        "1.0",
		GithubIssueURL: "https://github.com/org/repo/issues/1",
		JiraTicketURL:  "https://jira.org/browse/ISSUE-1",
		BranchType:     0,
	}

	result := n.String()
	require.NotEmpty(t, result)

	// Verify we can unmarshal the string back
	var deserialized notes.Notes
	err := json.Unmarshal([]byte(result), &deserialized)
	require.NoError(t, err)

	// Verify contents match
	require.Equal(t, n.Version, deserialized.Version)
	require.Equal(t, n.GithubIssueURL, deserialized.GithubIssueURL)
	require.Equal(t, n.JiraTicketURL, deserialized.JiraTicketURL)
	require.Equal(t, n.BranchType, deserialized.BranchType)
}
