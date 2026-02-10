package notes_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/qs/internal/notes"
)

func TestSerialize(t *testing.T) {
	tests := []struct {
		name           string
		githubIssueURL string
		jiraTicketURL  string
		branchType     notes.BranchType
		wantErr        bool
	}{
		{
			name:           "Basic serialization",
			githubIssueURL: "https://github.com/org/repo/issues/1",
			jiraTicketURL:  "https://jira.org/browse/ISSUE-1",
			branchType:     notes.BranchTypeDev,
			wantErr:        false,
		},
		{
			name:           "Empty URLs",
			githubIssueURL: "",
			jiraTicketURL:  "",
			branchType:     notes.BranchTypePr,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized, err := notes.Serialize(tt.githubIssueURL, tt.jiraTicketURL, tt.branchType, "")

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
			require.Equal(t, tt.githubIssueURL, n.GithubIssueURL)
			require.Equal(t, tt.jiraTicketURL, n.JiraTicketURL)
			require.Equal(t, tt.branchType, n.BranchType)
			require.NotEmpty(t, n.Version)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := notes.Deserialize(tt.input)

			if tt.wantErr {
				require.False(t, ok)
				require.Nil(t, got)
				return
			}

			require.True(t, ok)
			require.NotNil(t, got)
			require.Equal(t, tt.expected.Version, got.Version)
			require.Equal(t, tt.expected.GithubIssueURL, got.GithubIssueURL)
			require.Equal(t, tt.expected.JiraTicketURL, got.JiraTicketURL)
			require.Equal(t, tt.expected.BranchType, got.BranchType)
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
