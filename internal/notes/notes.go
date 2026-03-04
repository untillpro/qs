package notes

// Package notes provides functionality to serialize and deserialize Notes metadata structure.

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/voedger/voedger/pkg/goutils/logger"
)

// current version of the Notes struct
// This version is used to track changes in the Notes structure and ensure compatibility.
// It should be updated whenever there are changes to the structure or its fields.
var version = "1.0"

// supportedVersions lists all Notes versions that this binary can read.
// Deserialization fails with an error if the stored version is not in this list.
var supportedVersions = map[string]struct{}{
	"1.0": {},
}

// old plain-text format markers used before JSON notes were introduced
const (
	httpsPrefix = "https://"
)

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
	// Deprecated: use IssueURL. Kept for backward compatibility with old notes.
	GithubIssueURL string `json:"github_issue_url,omitempty"`
	// Deprecated: use IssueURL. Kept for backward compatibility with old notes.
	JiraTicketURL string     `json:"jira_ticket_url,omitempty"`
	BranchType    BranchType `json:"branch_type"` // Optional field to specify the type of branch (`dev` or `pr`)
	// Description is the original text from the dev branch creation (e.g., from `qs dev {some text}`)
	Description string `json:"description,omitempty"`
	// IssueURL is the URL of the issue associated with the branch (GitHub, Jira, or any other tracker).
	IssueURL string `json:"issue_url,omitempty"`
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
	issueURL string,
	branchType BranchType,
	description string,
) (string, error) {
	n := Notes{
		Version:     version,
		IssueURL:    issueURL,
		BranchType:  branchType,
		Description: description,
	}

	bytes, err := json.Marshal(n)
	if err != nil {
		return "", fmt.Errorf("failed to marshal notes: %w", err)
	}

	return string(bytes), nil
}

// ReadNotes reads notes from rawNotes, supporting all formats:
//   - JSON format (version 1.0) — pure JSON blob
//   - Old plain-text format — "Resolves issue" / "Resolves #" markers or plain title + URL
//
// Returns a *Notes struct populated from whichever format is detected.
// For old plain-text format, BranchType is set to BranchTypeDev.
func ReadNotes(rawNotes []string) (*Notes, error) {
	// Try JSON first
	n, err := Deserialize(rawNotes)
	if err == nil {
		return n, nil
	}

	// Fall back: scan lines for old plain-text format
	var description, issueURL string
	for _, line := range rawNotes {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "{") {
			// JSON blob that failed to deserialize — skip
			continue
		}
		if strings.Contains(line, httpsPrefix) {
			if issueURL == "" {
				issueURL = line
			}
		} else {
			if description == "" {
				description = line
			}
		}
	}

	if description == "" && issueURL == "" {
		return nil, fmt.Errorf("failed to read notes: %w", err)
	}

	return &Notes{
		BranchType:  BranchTypeDev,
		Description: description,
		IssueURL:    issueURL,
	}, nil
}

// Deserialize fetches JSON string object and tries to unmarshal it into Notes structure.
// Parameters:
// - notes: a slice of strings
// Returns:
// - a pointer to Notes structure if successful
// - error if JSON parsing fails or the stored version is not in supportedVersions
func Deserialize(notes []string) (*Notes, error) {
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
	if err := json.Unmarshal([]byte(jsonString.String()), &n); err != nil {
		return nil, fmt.Errorf("failed to unmarshal notes: %w", err)
	}

	if _, ok := supportedVersions[n.Version]; !ok {
		return nil, fmt.Errorf("unsupported notes version %q, please upgrade qs", n.Version)
	}

	return &n, nil
}
