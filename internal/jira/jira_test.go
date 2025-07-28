package jira

import (
	"testing"
)

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
