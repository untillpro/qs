# .augment/rules/qs-github-integration.md
type: manual

- Use GitHub CLI (gh) for all GitHub API operations
- Implement retry mechanisms with utils.Retry() for network calls
- Include proper error messages for GitHub authentication failures
- Follow the existing patterns in gitcmds/ package