package systrun

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/untillpro/qs/gitcmds"
)

func findBranchNameWithPrefix(repoPath, prefix string) (string, error) {
	repo, err := gitcmds.OpenGitRepository(repoPath)
	if err != nil {
		return "", err
	}

	branches, err := repo.Branches()
	if err != nil {
		return "", fmt.Errorf("failed to get branches: %w", err)
	}

	var foundBranch string
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		if strings.HasPrefix(ref.Name().Short(), prefix) {
			foundBranch = ref.Name().Short()
			return nil // Stop iteration once we find the branch
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("error iterating branches: %w", err)
	}

	if foundBranch == "" {
		return "", fmt.Errorf("no branch found with prefix '%s'", prefix)
	}

	return foundBranch, nil
}

func parseGithubIssueURL(issueURL string) (string, string, string, error) {
	// Extract repo owner, repo name, and issue number from the URL
	regExp := regexp.MustCompile(`https://github\.com/([^/]+)/([^/]+)/issues/(\d+)`)
	matches := regExp.FindStringSubmatch(issueURL)
	if matches == nil {
		return "", "", "", fmt.Errorf("invalid GitHub issue URL format: %s", issueURL)
	}

	repoOwner := matches[1] //nolint:revive
	repoName := matches[2]  //nolint:revive
	issueNum := matches[3]  //nolint:revive

	return repoOwner, repoName, issueNum, nil
}
