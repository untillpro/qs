package issue

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/internal/jira"
	"github.com/untillpro/qs/internal/notes"
	"github.com/untillpro/qs/utils"
)

type IssueType int

const (
	FreeForm IssueType = iota
	GitHub
	Jira
)

const (
	maximumBranchNameLength = 100
	issuePRTitlePrefix      = "Resolves issue"
	issueSign               = "Resolves #"
)

// IssueInfo holds parsed issue metadata.
// ID format depends on Type:
//   - GitHub: issue number (e.g. "42")
//   - Jira: ticket key (e.g. "AIR-270")
//   - FreeForm: empty
type IssueInfo struct {
	Type IssueType
	ID   string
	Text string
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

	// Build notes (common for GitHub and Jira when title is available)
	var githubURL, jiraURL string
	switch info.Type {
	case GitHub:
		githubURL = info.Text
	case Jira:
		jiraURL = info.Text
	}
	notesObj, err := notes.Serialize(githubURL, jiraURL, notes.BranchTypeDev, title)
	if err != nil {
		return "", nil, err
	}

	// Build comments
	switch info.Type {
	case GitHub:
		comment := issuePRTitlePrefix + " '" + title + "' "
		body := ""
		if len(title) > 0 {
			body = issueSign + info.ID + " " + title
		}
		comments = []string{comment, body, notesObj}
	case Jira:
		if notesObj != "" {
			comments = append(comments, notesObj)
		}
		comments = append(comments, "["+info.ID+"] "+title)
		comments = append(comments, info.Text)
	default:
		comments = append(comments, title)
		comments = append(comments, notesObj)
	}

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
	parts := strings.Split(info.Text, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid issue URL format: %s", info.Text)
	}

	repoURL := strings.Split(info.Text, "/issues/")[0]
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
	if strings.Contains(text, "/issues/") {
		segments := strings.Split(text, "/")
		return IssueInfo{Type: GitHub, Text: text, ID: segments[len(segments)-1]}, nil
	}
	if id, ok := jira.GetJiraTicketIDFromArgs(args...); ok {
		return IssueInfo{Type: Jira, Text: text, ID: id}, nil
	}
	return IssueInfo{Type: FreeForm, Text: text}, nil
}
