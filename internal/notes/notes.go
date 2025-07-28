package notes

// Package notes provides functionality to serialize and deserialize Notes metadata structure.

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/untillpro/goutils/logger"
)

// current version of the Notes struct
// This version is used to track changes in the Notes structure and ensure compatibility.
// It should be updated whenever there are changes to the structure or its fields.
var version = "1.0"

type BranchType int

const (
	BranchTypeUnknown BranchType = iota
	BranchTypeDev
	BranchTypePr
)

func (bt BranchType) String() string {
	switch bt {
	case BranchTypeUnknown:
		return "unknown"
	case BranchTypeDev:
		return "dev"
	case BranchTypePr:
		return "pr"
	default:
		return "unknown"
	}
}

// Notes represents the structure for storing metadata related to branch.
type Notes struct {
	// Version indicates the version of the Notes structure.
	Version string `json:"version"`
	// GithubIssueURL is the URL of the GitHub issue associated with the branch.
	GithubIssueURL string `json:"github_issue_url"`
	// JiraTicketURL is the URL of the Jira ticket associated with the branch.
	JiraTicketURL string     `json:"jira_ticket_url"`
	BranchType    BranchType `json:"branch_type"` // Optional field to specify the type of branch (`dev` or `pr`)
}

// Serialize is a function for serializing given notes field into a JSON string representation.
// Returns a JSON string representation of the notes
func (nt *Notes) String() string {
	bytes, err := json.Marshal(nt)
	if err != nil {
		logger.Verbose(fmt.Errorf("failed to marshal notes: %w", err))

		os.Exit(1)
	}

	return string(bytes)
}

// Serialize is a function for serializing given notes field into a JSON string representation.
// Returns a JSON string representation of the notes and an error if serialization fails.
func Serialize(
	githubIssueURL string,
	jiraTicketURL string,
	branchType BranchType,
) (string, error) {
	n := Notes{
		Version:        version,
		GithubIssueURL: githubIssueURL,
		JiraTicketURL:  jiraTicketURL,
		BranchType:     branchType,
	}

	bytes, err := json.Marshal(n)
	if err != nil {
		return "", fmt.Errorf("failed to marshal notes: %w", err)
	}

	return string(bytes), nil
}

// Deserialize fetches JSON string object and tries to unmarshal it into Notes structure.
// Parameters:
// - notes: a slice of strings
// Returns:
// - a pointer to Notes structure if successful
// - true if successful, false otherwise
func Deserialize(notes []string) (*Notes, bool) {
	jsonString := strings.Builder{}
	jsonStringStarted := false
	for _, note := range notes {
		i := 0
		if !jsonStringStarted {
			i = strings.Index(note, "{")
			if i >= 0 {
				jsonStringStarted = true
			}
		}

		if jsonStringStarted {
			j := strings.LastIndex(note, "}")
			if j >= 0 {
				jsonString.WriteString(note[i : j+1])
				break
			} else {
				jsonString.WriteString(note[i:])
			}
		}

	}

	var n Notes
	if err := json.Unmarshal([]byte(jsonString.String()), &n); err == nil {
		return &n, true
	}

	return nil, false
}
