---
registered_at: 2026-03-03T11:49:11Z
change_id: 2603031149-customer-display-size-modifiers
baseline: 7705a2db7ce4bd31e8c19ea944623d0733824b93
---

# Change request: Support description with URL in qs dev input

## Terminology

- Fetchable URL: a URL from which the issue title can be fetched via API (GitHub issues, Jira tickets)
- Non-fetchable URL: any other URL (e.g. ProjectKaiser URLs)

## Why

The qs dev command currently processes input as either a URL or free-form text. Users need the ability to provide both a description and a non-fetchable URL together, either on the same line or on separate lines.

## What

Enhance qs dev to accept the following input formats:

Format 1 - Text + non-fetchable URL on same line:
```text
TBlrBill.UpdateCustomerDisplays: does not display size modifiers https://dev.untill.com/projects/#!763090
```

Format 2 - Text + non-fetchable URL on separate lines:
```text
TBlrBill.UpdateCustomerDisplays: does not display size modifiers
https://dev.untill.com/projects/#!763090
```

Format 3 - Fetchable URL only (existing behavior):
```text
https://github.com/untillpro/qs/issues/123
```

Format 4 - Text only (existing behavior):
```text
TBlrBill.UpdateCustomerDisplays: does not display size modifiers
```

Other inputs are errors, e.g. non-fetchable URL without text.

The command should:

- Parse input to extract text and URL separately
- Always populate URL field when URL is present in input
- For fetchable URLs: fetch title from API for branch naming (existing behavior)
- For non-fetchable URLs: use text for branch naming (require text)
- For text-only input: use text for branch naming (no URL)
- Store all URLs in branch notes (fetchable URLs are already stored; non-fetchable URLs are new)
- Maintain backward compatibility with existing formats
