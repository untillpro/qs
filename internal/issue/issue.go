package issue

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/untillpro/qs/internal/jira"
)

type IssueType int

const (
	FreeForm IssueType = iota
	GitHub
	Jira
)

type IssueInfo struct {
	Type   IssueType
	URL    string
	ID     string
	Number int
}

func ParseIssueFromArgs(wd string, args ...string) (IssueInfo, error) {
	if len(args) != 1 {
		return IssueInfo{}, nil
	}
	url := args[0]
	if strings.Contains(url, "/issues/") {
		if err := checkGitHubIssue(wd, url); err != nil {
			return IssueInfo{}, fmt.Errorf("invalid GitHub issue link: %w", err)
		}
		segments := strings.Split(url, "/")
		num, err := strconv.Atoi(segments[len(segments)-1])
		if err != nil {
			return IssueInfo{}, fmt.Errorf("failed to convert issue number from string to int: %w", err)
		}
		return IssueInfo{Type: GitHub, URL: url, ID: segments[len(segments)-1], Number: num}, nil
	}
	if id, ok := jira.GetJiraTicketIDFromArgs(args...); ok {
		return IssueInfo{Type: Jira, URL: url, ID: id}, nil
	}
	return IssueInfo{}, nil
}

func checkGitHubIssue(wd, issueURL string) error {
	cmd := exec.Command("gh", "issue", "view", "--json", "title,state", issueURL)
	cmd.Dir = wd
	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("failed to check issue link: %w", err)
	}
	return nil
}
