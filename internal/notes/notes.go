package notes

// Package notes provides functionality to serialize and deserialize Notes metadata structure.

import (
	"encoding/json"
	"fmt"
	"os"
)

// current version of the Notes struct
// This version is used to track changes in the Notes structure and ensure compatibility.
// It should be updated whenever there are changes to the structure or its fields.
var version = "1.0"

type BranchType string

const (
	BranchTypeDev BranchType = "dev"
	BranchTypePr             = "pr"
)

// Notes represents the structure for storing metadata related to branch.
type Notes struct {
	// Version indicates the version of the Notes structure.
	Version string `json:"version"`
	// GithubIssueURL is the URL of the GitHub issue associated with the branch.
	GithubIssueURL string `json:"github_issue_url"`
	// JiraTicketURL is the URL of the Jira ticket associated with the branch.
	JiraTicketURL string `json:"jira_ticket_url"`
	BranchType    string `json:"branch_type"` // Optional field to specify the type of branch (e.g., feature, bugfix)
}

// Serialize is a function for serializing given notes field into a JSON string representation.
// Returns a JSON string representation of the notes
func Serialize(
	githubIssueURL string,
	jiraTicketURL string,
	branchType BranchType,
) string {
	n := Notes{
		Version:        version,
		GithubIssueURL: githubIssueURL,
		JiraTicketURL:  jiraTicketURL,
		BranchType:     string(branchType),
	}

	bytes, err := json.Marshal(n)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to marshal notes: %w", err)
		os.Exit(1)
		return ""
	}

	return string(bytes)
}

// Deserialize fetches the first valid Notes structure from a slice of strings.
func Deserialize(notes []string) *Notes {
	for _, note := range notes {
		var n Notes
		if err := json.Unmarshal([]byte(note), &n); err == nil {
			return &n
		}
	}

	return nil
}
