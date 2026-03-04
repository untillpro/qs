package issue

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/qs/internal/jira"
	"github.com/untillpro/qs/internal/notes"
	"github.com/untillpro/qs/utils"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

type IssueType int

const (
	FreeForm IssueType = iota
	GitHub
	Jira
)

const (
	maximumBranchNameLength = 100
)

// IssueInfo holds parsed issue metadata.
// ID format depends on Type:
//   - GitHub: issue number (e.g. "42")
//   - Jira: ticket key (e.g. "AIR-270")
//   - FreeForm: empty
type IssueInfo struct {
	Type IssueType
	ID   string
	// URL holds the URL extracted from the input (fetchable or non-fetchable).
	URL string
	// Text holds the user-provided description text (URL excluded).
	Text string
}

// extractURLFromText extracts the first URL (http:// or https://) from text.
// Returns the extracted URL and the remaining text with the URL removed and trimmed.
// If no URL is found, returns empty string and the full text as remainingText.
func extractURLFromText(text string) (url string, remainingText string) {
	re := regexp.MustCompile(`https?://[^\s]+`)
	loc := re.FindStringIndex(text)
	if loc == nil {
		return "", text
	}
	url = text[loc[0]:loc[1]]
	remaining := strings.TrimSpace(text[:loc[0]] + text[loc[1]:])
	return url, remaining
}

// ExtractIDFromURL extracts an ID from the last slash-separated segment of a URL.
// Strips any leading '#' and '!' characters (e.g. "#!763090" -> "763090").
// Returns empty string if the URL has no usable segment.
func ExtractIDFromURL(rawURL string) string {
	segments := strings.Split(rawURL, "/")
	return strings.TrimLeft(segments[len(segments)-1], "#!")
}

func BuildDevBranchName(info IssueInfo) (devBranchName string, comments []string, err error) {
	var title string

	switch info.Type {
	case GitHub:
		title, err = fetchGithubIssueTitle(info)
	case Jira:
		title, _, err = jira.GetJiraIssueTitle("", info.ID)
	default:
		title = info.Text
	}
	if err != nil {
		return "", nil, err
	}

	// Convert issue title to kebab-style branch name
	devBranchName = titleToKebabWithPrefix(info.ID, title)
	devBranchName = utils.CleanArgFromSpecSymbols(devBranchName)

	// Build notes with unified IssueURL for all issue types.
	notesObj, err := notes.Serialize(info.URL, notes.BranchTypeDev, title)
	if err != nil {
		return "", nil, err
	}

	comments = []string{notesObj}

	devBranchName += "-dev"

	return devBranchName, comments, nil
}

// titleToKebabWithPrefix converts an issue title into a kebab-case branch name prefixed with the issue ID.
func titleToKebabWithPrefix(id, title string) string {
	kebabTitle := strings.ToLower(title)
	kebabTitle = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(kebabTitle, "-")
	kebabTitle = strings.Trim(kebabTitle, "-")

	branchName := fmt.Sprintf("%s-%s", id, kebabTitle)
	if len(branchName) > maximumBranchNameLength {
		branchName = branchName[:maximumBranchNameLength]
	}
	return branchName
}

// fetchGithubIssueTitle fetches the issue title from GitHub.
func fetchGithubIssueTitle(info IssueInfo) (string, error) {
	parts := strings.Split(info.URL, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid issue URL format: %s", info.URL)
	}

	repoURL := strings.Split(info.URL, "/issues/")[0]
	urlParts := strings.Split(repoURL, "/")
	if len(urlParts) < 5 { //nolint:revive
		return "", fmt.Errorf("invalid GitHub URL format: %s", repoURL)
	}
	owner := urlParts[3] //nolint:revive
	repo := urlParts[4]  //nolint:revive

	stdout, stderr, err := new(exec.PipedExec).
		Command("gh", "issue", "view", info.ID, "--repo", fmt.Sprintf("%s/%s", owner, repo), "--json", "title").
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)
		if len(stderr) > 0 {
			return "", errors.New(stderr)
		}
		return "", fmt.Errorf("failed to get issue title: %w", err)
	}
	logger.Verbose(stdout)

	var issueData struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(stdout), &issueData); err != nil {
		return "", fmt.Errorf("failed to parse issue data: %w", err)
	}

	return issueData.Title, nil
}

func ParseIssueFromArgs(args ...string) (IssueInfo, error) {
	text := args[0] // protected by caller side
	url, remainingText := extractURLFromText(text)

	if url != "" {
		if strings.Contains(url, "/issues/") {
			segments := strings.Split(url, "/")
			return IssueInfo{Type: GitHub, URL: url, Text: remainingText, ID: segments[len(segments)-1]}, nil
		}
		if id, ok := jira.GetJiraTicketIDFromArgs(url); ok {
			return IssueInfo{Type: Jira, URL: url, Text: remainingText, ID: id}, nil
		}
		// Non-fetchable URL: text is required for branch naming
		if remainingText == "" {
			return IssueInfo{}, errors.New("text is required when URL is non-fetchable")
		}
		return IssueInfo{Type: FreeForm, URL: url, Text: remainingText, ID: ExtractIDFromURL(url)}, nil
	}
	return IssueInfo{Type: FreeForm, Text: text}, nil
}
