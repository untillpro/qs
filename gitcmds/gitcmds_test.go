package gitcmds

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	notesPkg "github.com/untillpro/qs/internal/notes"
)

func TestGetNotes(t *testing.T) {
	// Test for PK types
	s1 := "Permanent support for Peter,  Pascal, and customers "
	s2 := " https://dev.untill.com/projects/#!361164  "
	notes := []string{s1, s2}
	title, url := GetNoteAndURL(notes)
	assert.Equal(t, "Permanent support for Peter,  Pascal, and customers", title)
	assert.Equal(t, "https://dev.untill.com/projects/#!361164", url)

	notes = []string{s2, s1}
	title, url = GetNoteAndURL(notes)
	assert.Equal(t, "Permanent support for Peter,  Pascal, and customers", title)
	assert.Equal(t, "https://dev.untill.com/projects/#!361164", url)

	// Test for GH Issue types
	s1 = "Resolves issue #324 My Best problem ever"
	s2 = "Resolves #324"
	notes = []string{s1, s2}
	title, url = GetNoteAndURL(notes)
	assert.Equal(t, "Resolves issue #324 My Best problem ever", title)
	assert.Empty(t, "", url)

	// JSON blob must not be included in note or url
	s1 = "Some description"
	s2 = `{"version":"1.0","github_issue_url":"","jira_ticket_url":"","branch_type":2,"description":"Some description"}`
	notes = []string{s1, s2}
	title, url = GetNoteAndURL(notes)
	assert.Equal(t, "Some description", title)
	assert.Empty(t, url)

	// JSON blob must not be included even when it contains https inside field values
	s1 = "My feature"
	s2 = "https://dev.untill.com/projects/#!763090"
	s3 := `{"version":"1.0","issue_url":"https://dev.untill.com/projects/#!763090","branch_type":1,"description":"My feature"}`
	notes = []string{s1, s2, s3}
	title, url = GetNoteAndURL(notes)
	assert.Equal(t, "My feature", title)
	assert.Equal(t, "https://dev.untill.com/projects/#!763090", url)
}

func TestGetBody(t *testing.T) {
	// PK type: returns the description
	s1 := "Permanent support for Peter, Pascal, and customers "
	s2 := " https://dev.untill.com/projects/#!361164  "
	notes := []string{s1, s2}
	body := GetBodyFromNotes(notes)
	require.Equal(t, "Permanent support for Peter, Pascal, and customers", body)

	// GH Issue old format: returns the first description line
	s1 = "Resolves issue #324 My Best problem ever"
	s2 = " Resolves #324  "
	notes = []string{s1, s2}
	body = GetBodyFromNotes(notes)
	require.Equal(t, "Resolves issue #324 My Best problem ever", body)

	// JSON format: returns description field from JSON
	jsonNote := `{"version":"1.0","github_issue_url":"","jira_ticket_url":"","branch_type":1,"description":"My feature"}`
	notes = []string{jsonNote}
	body = GetBodyFromNotes(notes)
	require.Equal(t, "My feature", body)
}

func TestDeserializeVersionCheck(t *testing.T) {
	tests := []struct {
		name    string
		notes   []string
		wantErr bool
		wantBT  notesPkg.BranchType
	}{
		{
			name:    "supported version",
			notes:   []string{`{"version":"1.0","github_issue_url":"","jira_ticket_url":"","branch_type":1}`},
			wantErr: false,
			wantBT:  notesPkg.BranchTypeDev,
		},
		{
			name:    "unsupported version",
			notes:   []string{`{"version":"9.9","github_issue_url":"","jira_ticket_url":"","branch_type":1}`},
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			notes:   []string{"not json at all"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := notesPkg.Deserialize(tt.notes)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, n)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, n)
			assert.Equal(t, tt.wantBT, n.BranchType)
		})
	}
}

func TestNormalizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple branch name",
			input:    "feature-123",
			expected: "feature-123",
		},
		{
			name:     "branch with spaces",
			input:    "feature 123 test",
			expected: "feature-123-test",
		},
		{
			name:     "branch with invalid characters",
			input:    "feature~123^test:branch?name",
			expected: "feature-123-test-branch-name",
		},
		{
			name:     "branch with double dots",
			input:    "feature..123",
			expected: "feature-123",
		},
		{
			name:     "branch with leading dot",
			input:    ".feature-123",
			expected: "feature-123",
		},
		{
			name:     "branch with trailing slash",
			input:    "feature-123/",
			expected: "feature-123",
		},
		{
			name:     "branch with leading and trailing dashes",
			input:    "-feature-123-",
			expected: "feature-123",
		},
		{
			name:     "branch with consecutive dashes",
			input:    "feature---123-",
			expected: "feature-123",
		},
		{
			name:     "branch with uppercase",
			input:    "Feature-123-Test",
			expected: "Feature-123-Test",
		},
		{
			name:     "branch with special characters",
			input:    "feature[123]*test",
			expected: "feature-123-test",
		},
		{
			name:     "branch with backslash",
			input:    "feature\\123",
			expected: "feature-123",
		},
		{
			name:     "branch with tabs and newlines",
			input:    "feature\t123\ntest",
			expected: "feature-123-test",
		},
		{
			name:     "complex branch name",
			input:    "Fix: Issue #123 - Add new feature!",
			expected: "Fix-Issue-123-Add-new-feature",
		},
		{
			name:     "branch with underscores at edges",
			input:    "_feature-123_",
			expected: "feature-123",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only invalid characters",
			input:    "~~~^^^:::???",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeBranchName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
