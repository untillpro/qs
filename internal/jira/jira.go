package jira

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
)

// GetJiraTicketIDFromArgs retrieves a JIRA ticket ID from the provided arguments.
// parameters:
// - args: A variable number of string arguments that may contain JIRA issue URLs.
// returns:
// - jiraTicketID: The JIRA ticket ID if found.
// - ok: A boolean indicating whether a JIRA ticket ID was found.
func GetJiraTicketIDFromArgs(args ...string) (jiraTicketID string, ok bool) {
	// Define a regular expression to match JIRA issue URLs
	// The regex matches URLs like "https://<subdomain>.atlassian.net/browse/<ISSUE-KEY>"
	re := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.atlassian\.net/browse/([A-Z]+-[A-Z0-9-]+)`)
	for _, arg := range args {
		// Check if the argument matches the pattern
		if matches := re.FindStringSubmatch(arg); matches != nil {
			// Return the issue key (group 1) and true
			return matches[1], true
		}
	}

	// No matching argument was found
	return "", false
}

// GetJiraIssueTitle retrieves the name of a JIRA issue based on its ticket ID or URL.
// parameters:
// - ticketURL: The URL of the JIRA ticket (optional).
// - ticketID: The ID of the JIRA ticket (optional).
func GetJiraIssueTitle(ticketURL, ticketID string) (string, string, error) {
	// Validate the issue key
	if ticketID == "" {
		var ok bool
		ticketID, ok = GetJiraTicketIDFromArgs(ticketURL)
		if !ok {
			return "", ticketID, errors.New("error: ticketID or ticketURL is required")
		}
	}

	// Retrieve API token and email from environment variables
	apiToken := os.Getenv("JIRA_API_TOKEN")
	if apiToken == "" {
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println("Error: JIRA API token not found. Please set environment variable JIRA_API_TOKEN.")
		fmt.Println("            Jira API token can generate on this page:")
		fmt.Println("          https://id.atlassian.com/manage-profile/security/api-tokens           ")
		fmt.Println("--------------------------------------------------------------------------------")

		return "", ticketID, errors.New("error: JIRA API token not found")
	}
	var (
		email string
		err   error
	)

	email = os.Getenv("JIRA_EMAIL")
	if email == "" {
		return "", ticketID, errors.New("error: please export JIRA_EMAIL")
	}

	// Build the request URL
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", jiraDomain, ticketID)

	// Create HTTP client and request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", ticketID, err
	}
	req.SetBasicAuth(email, apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", ticketID, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ticketID, err
	}
	// Issue does not exist or you do not have permission to see it.

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return "", ticketID, ErrJiraIssueNotFoundOrInsufficientPermission
		}

		return "", ticketID, err
	}

	var result struct {
		Fields struct {
			Summary string `json:"summary"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", ticketID, fmt.Errorf("error parsing JSON response: %w", err)
	}

	// Check if the summary field exists
	if result.Fields.Summary == "" {
		return "", ticketID, nil
	}

	return result.Fields.Summary, ticketID, nil
}
