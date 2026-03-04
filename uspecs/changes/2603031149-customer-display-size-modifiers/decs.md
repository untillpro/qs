# Decisions: Support description with URL in qs dev input

## Terminology

- Fetchable URL: a URL from which the issue title can be fetched via API (GitHub issues, Jira tickets)
- Non-fetchable URL: any other URL (e.g. ProjectKaiser URLs)

## URL detection and extraction strategy

Use regex pattern to detect and extract URLs from input text (confidence: high).

Rationale: URLs have well-defined patterns (http/https schemes). A regex can reliably identify and extract URLs from mixed text, allowing separation of description and URL regardless of whether they are on the same line or different lines.

Implementation approach:

- Apply regex pattern to match URLs (e.g., `https?://[^\s]+`)
- Extract matched URL(s) from input
- Remove URL from input to get description text
- Trim whitespace from description

Alternatives:

- Split by whitespace and check each token (confidence: medium)
  - Simpler but may fail if URL contains spaces or special characters
  - Harder to handle URLs embedded in text
- Split by newline first, then check if line is URL (confidence: medium)
  - Works for multi-line format but fails for single-line format with description and URL

## Handling multiple URLs in input

Use the first detected URL only (confidence: high).

Rationale: Most common use case is single issue URL. Using first URL is predictable and simple. Additional URLs in description text are likely part of the description context, not issue references.

Alternatives:

- Use the last URL (confidence: low)
  - Less intuitive, users expect first URL to be the issue reference
- Collect all URLs and store in notes (confidence: medium)
  - More complex, unclear which URL is the primary issue reference
- Return error if multiple URLs detected (confidence: low)
  - Too restrictive, prevents valid use cases where description mentions other URLs

## URL type detection order

Check for URL first, then apply existing issue type detection (confidence: high).

Rationale: Maintains backward compatibility with existing fetchable URL detection logic. The existing `ParseIssueFromArgs` function already handles URL type detection correctly.

Flow:

- Extract URL from input (if present)
- Extract text from input (remaining text after URL removal)
- If URL exists, pass URL to existing `ParseIssueFromArgs` logic for type detection (fetchable or non-fetchable)
- If no URL, pass text to `ParseIssueFromArgs` as FreeForm
- Populate both URL and Text fields in IssueInfo

Alternatives:

- Detect issue type from URL pattern before extraction (confidence: medium)
  - Duplicates existing logic in `ParseIssueFromArgs`
- Always treat extracted URL as non-fetchable (confidence: low)
  - Loses fetchable URL handling

## IssueInfo structure changes

Add URL and Text fields to IssueInfo struct with clear separation of concerns (confidence: high).

Rationale: Each field has a single, well-defined purpose. URL always holds the URL (if present), Text always holds the user-provided text. This eliminates overloading and makes validation straightforward. For fetchable URLs, title is fetched from API. For non-fetchable URLs, Text is required for branch naming. For text-only input, Text is used for branch naming.

Structure:
```go
type IssueInfo struct {
    Type IssueType
    ID   string
    URL  string // Always set from params when URL exists in input
    Text string // User-provided text
}
```

Usage:

- Fetchable URL only: URL populated, Text empty, title fetched from API for branch naming
- Fetchable URL + Text: URL and Text populated, Text ignored, title fetched from API for branch naming
- Non-fetchable URL + Text: URL and Text populated, Text used for branch naming, URL stored in notes
- Text only (no URL): URL empty, Text populated, Text used for branch naming
- Error case: non-fetchable URL without Text (text required for branch naming)

Alternatives:

- Keep Text field for URL, add separate Text field (confidence: medium)
  - Overloaded semantics, Text field meaning changes based on context
  - Less clear which field to use in different scenarios
- Use single Text field for both URL and text (confidence: low)
  - Current approach, requires parsing Text field every time to determine if it's URL or text

## Backward compatibility handling

Parse input to populate URL and Text fields separately based on content (confidence: high).

Rationale: Maintains existing behavior for all current input formats. No breaking changes to existing functionality. Clear validation rules for new formats.

Behavior:

- Fetchable URL only: URL = URL, Text = empty, title fetched from API for branch naming, URL stored in notes (existing behavior)
- Text only: URL = empty, Text = text, text used for branch naming, no URL in notes (existing behavior)
- Non-fetchable URL + Text: URL = URL, Text = text, text used for branch naming, URL stored in notes (NEW)
- Non-fetchable URL only: ERROR - text required for branch naming
- Fetchable URL + Text: URL = URL, Text = text (optional, ignored), title fetched from API for branch naming, URL stored in notes

Validation:

- Fetchable URL type: Text is optional (title fetched from API)
- Non-fetchable URL type: Text is required (error if missing)

Alternatives:

- Require explicit format indicator (confidence: low)
  - Breaking change, adds complexity for users
- Allow non-fetchable URL only, use URL as fallback text (confidence: medium)
  - Less user-friendly branch names for non-fetchable URLs
