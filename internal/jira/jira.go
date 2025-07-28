package jira

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"

	"github.com/untillpro/qs/internal/helper"
	"github.com/untillpro/qs/internal/notes"
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

// GetJiraBranchName generates a branch name based on a JIRA issue URL in the arguments.
// If a JIRA URL is found, it generates a branch name in the format "<ISSUE-KEY>-<cleaned-description>".
// Additionally, it generates comments in the format "[<ISSUE-KEY>] <original-line>".
func GetJiraBranchName(args ...string) (branch string, comments []string, err error) {
	comments = make([]string, 0, len(args)+1) // 1 for json notes
	for _, arg := range args {
		jiraTicketID, ok := GetJiraTicketIDFromArgs(arg)
		if ok {
			var brName string
			issueName, err := GetJiraIssueName("", jiraTicketID)
			if err != nil {
				return "", nil, err
			}

			if issueName == "" {
				branch, _, err = helper.GetBranchName(false, args...)
				if err != nil {
					return "", nil, err
				}
			} else {
				jiraTicketURL := arg // Full JIRA ticket URL
				// Prepare new notes
				notesObj, err := notes.Serialize("", jiraTicketURL, notes.BranchTypeDev)
				if err != nil {
					return "", nil, err
				}
				comments = append(comments, notesObj)
				brName, _, err = helper.GetBranchName(false, issueName)
				if err != nil {
					return "", nil, err
				}
				branch = jiraTicketID + "-" + brName
			}
			comments = append(comments, "["+jiraTicketID+"] "+issueName)
		}
	}
	// Add suffix "-dev" for a dev branch
	branch += "-dev"
	comments = append(comments, args...)

	return branch, comments, nil
}

// GetJiraIssueName retrieves the name of a JIRA issue based on its ticket ID or URL.
// parameters:
// - ticketURL: The URL of the JIRA ticket (optional).
// - ticketID: The ID of the JIRA ticket (optional).
func GetJiraIssueName(ticketURL, ticketID string) (name string, err error) {
	// Validate the issue key
	if ticketID == "" {
		var ok bool
		ticketID, ok = GetJiraTicketIDFromArgs(ticketURL)
		if !ok {
			return "", errors.New("error: ticketID or ticketURL is required")
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

		return "", errors.New("error: JIRA API token not found")
	}
	var email string
	email = os.Getenv("JIRA_EMAIL")
	if email == "" {
		email, err = helper.GetUserEmail() // Replace with your email
	}
	if err != nil {
		return "", err
	}
	if email == "" {
		return "", errors.New("error: please export JIRA_EMAIL")
	}
	fmt.Println("User email: ", email)

	// Build the request URL
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", jiraDomain, ticketID)

	// Create HTTP client and request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(email, apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", err
	}

	var result struct {
		Fields struct {
			Summary string `json:"summary"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error parsing JSON response: %w", err)
	}

	// Check if the summary field exists
	if result.Fields.Summary == "" {
		return "", nil
	}

	return result.Fields.Summary, nil
}
