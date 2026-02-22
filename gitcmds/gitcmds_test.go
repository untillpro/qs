package gitcmds

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
}

func TestGetBody(t *testing.T) {
	// Test for PK types
	s1 := "Permanent support for Peter, Pascal, and customers "
	s2 := " https://dev.untill.com/projects/#!361164  "
	notes := []string{s1, s2}
	body := GetBodyFromNotes(notes)
	require.Empty(t, body)

	// Test for GH Issue types
	s1 = "Resolves issue #324 My Best problem ever"
	s2 = " Resolves #324  "
	notes = []string{s1, s2}
	body = GetBodyFromNotes(notes)
	require.Equal(t, "Resolves #324", body)
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
