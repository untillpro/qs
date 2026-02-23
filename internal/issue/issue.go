package issue

import (
	"fmt"
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
	url := args[0] // protected by caller side
	if strings.Contains(url, "/issues/") {
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
	return IssueInfo{Type: FreeForm}, nil
}
