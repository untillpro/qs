# Quick Source Management Tool (qs)

A powerful Git workflow automation tool that simplifies repository management, branch operations, and GitHub integration.

## Features

- 🚀 **Streamlined Git workflows** - Automate common Git operations
- 🔄 **Fork management** - Easy repository forking and upstream configuration
- 🌿 **Branch lifecycle** - Create, manage, and clean up development branches
- 📝 **Pull request automation** - Create and manage PRs with GitHub integration
- 🎯 **Issue integration** - Link branches to GitHub issues and Jira tickets
- 🔄 **Sync operations** - Smart download/upload with conflict resolution
- 🛡️ **Safety checks** - Prevent large commits and validate operations
- 🔁 **Retry mechanisms** - Robust network operations with automatic retries
- 🧪 **System tests** - Comprehensive test suite with real GitHub integration

## Installation Prerequisites

### Required Tools

#### GitHub CLI (gh) - version > 2.27
- **Windows**: `scoop install main/gh` ([scoop.sh](https://scoop.sh/#/apps?q=gh))
- **macOS**: `brew install gh`
- **Linux**: Follow [GitHub CLI installation guide](https://github.com/cli/cli)

#### Git
- **Windows**: Ensure `$Git\usr\bin` is in PATH (provides Unix utilities: `grep`, `sed`, `jq`, `gawk`, etc.)
- **macOS**: `brew install git gawk jq` (gawk and jq required for qs operations)
- **Linux**: Usually pre-installed, ensure `gawk` and `jq` are available

#### Platform-specific Dependencies
- **Linux**: `sudo apt install xclip` (for clipboard operations)
- **macOS**: `brew install gawk jq` (if not already installed)

## Installation

```bash
go install github.com/untillpro/qs@latest
```

## Quick Start

`qs` automatically detects your workflow mode:
- **Single Remote Mode**: When you have direct push access (no upstream remote)
- **Fork Mode**: When contributing to external projects (with upstream remote)

```bash
# For repositories with direct access (single remote mode)
qs dev "feature-name"      # Create branch - no fork needed!
qs u -m "commit message"   # Upload changes
qs pr                      # Create PR to same repository

# For external projects (fork mode)
qs fork                    # Fork repository and configure remotes
qs dev "feature-name"      # Create branch in your fork
qs u -m "commit message"   # Upload changes
qs pr                      # Create PR to upstream

# Common commands (work in both modes)
qs d                       # Download changes (smart sync)
qs                         # Show repository status
```

## Command Reference

### Core Commands

#### Repository Status
```bash
qs                         # Show Git status of current repository
qs -v, --verbose          # Enable verbose output for all operations
qs -h, --help             # Show help information
```

#### Repository Management
```bash
qs fork                    # Fork repository to your account and configure upstream
                          # - Creates fork on GitHub
                          # - Configures origin → fork, upstream → original
                          # - Sets up proper remote tracking
```

#### Branch Operations
```bash
qs dev [branch-name]       # Create development branch
                          # - Auto-detects workflow mode (fork vs single remote)
                          # - Auto-detects branch name from clipboard
                          # - Supports GitHub issue URLs
                          # - Supports Jira ticket URLs
                          # - Links branch to issues automatically
                          # - Works with or without upstream remote

qs dev -d                  # Delete merged development branches
                          # - Removes local and remote branches
                          # - Only deletes merged branches
                          # - Cleans up tracking references

qs dev -i, --ignore-hook   # Create branch without large file hooks
```

#### Sync Operations
```bash
qs d                       # Download (smart sync from remotes)
                          # - Fetches from origin with --prune
                          # - Pulls notes from origin
                          # - Merges origin/main → main
                          # - Merges remote tracking branch (if exists)
                          # - Pulls from upstream (if configured)

qs u [-m "message"]        # Upload changes
                          # - Stages all changes
                          # - Creates commit with message
                          # - Pushes to remote
                          # - Sets up tracking branch (first time only)
                          # - Uses clipboard for commit message if no -m flag
```

#### Pull Request Management
```bash
qs pr                      # Create pull request
                          # - Creates PR from current dev branch
                          # - Links to GitHub issues (if branch is linked)
                          # - Converts dev branch to PR branch
                          # - Deletes dev branch after PR creation

qs pr -d, --draft          # Create draft pull request
```

#### Utility Commands
```bash
qs r                       # Create release (opens release interface)
qs g                       # Open Git GUI
qs version                 # Show current qs version
qs upgrade                 # Show upgrade command
```

### Global Flags
```bash
-C, --change-dir <dir>     # Change to directory before running command
-v, --verbose              # Enable verbose output
-h, --help                 # Show help
```

## Integration Features

### GitHub Integration

#### Issue Linking
```bash
# Create branch from GitHub issue URL
qs dev https://github.com/owner/repo/issues/123

# Branch will be automatically named and linked to the issue
# PR creation will reference the issue
```

#### Clipboard Integration
- `qs dev` automatically reads branch names from clipboard
- `qs u` uses clipboard content for commit messages (if no -m flag)
- Supports GitHub issue URLs, Jira tickets, and custom text

### Jira Integration

#### Setup
Set environment variables for Jira integration:
```bash
export JIRA_EMAIL="your-email@company.com"
export JIRA_API_TOKEN="your-jira-api-token"
```

Generate Jira API token: [Atlassian API Tokens](https://id.atlassian.com/manage-profile/security/api-tokens)

#### Usage
```bash
# Create branch from Jira ticket
qs dev https://untill.atlassian.net/browse/AIR-270

# This will:
# - Create branch named after the ticket
# - Link branch to Jira ticket
# - Include ticket reference in PR
```

#### Error Handling
- If `JIRA_EMAIL` is not set, qs tries to use Git user email
- Missing `JIRA_API_TOKEN` shows helpful error with setup instructions
## Safety Features

### Commit Size Limits
Automatic pre-commit hooks prevent large commits:

- **Total file size**: 100,000 bytes (~100KB) for all files combined
- **Number of files**: 200 files maximum
- **File exclusions**: Files with `.wasm` extension are excluded from size calculations

### Commit Message Validation
- Minimum commit message length: 8 characters
- Interactive prompts for short messages on main/master branches
- Clipboard integration for commit messages

### Repository State Checks
- Prevents operations on uncommitted changes (when unsafe)
- Validates Git repository state before operations
- Checks for proper remote configuration
- Verifies branch tracking setup

## Configuration

### Environment Variables

#### Core Configuration
```bash
# Skip version checks (useful for CI/CD)
export QS_SKIP_QS_VERSION_CHECK=true
```

#### Retry Configuration
```bash
# Network operation retry settings
export QS_MAX_RETRIES=5                    # Maximum retry attempts (default: 3)
export QS_RETRY_DELAY_SECONDS=3            # Initial delay between retries (default: 2)
export QS_MAX_RETRY_DELAY_SECONDS=60       # Maximum delay between retries (default: 30)

# GitHub CLI timeout
export GH_TIMEOUT_MS=2000                  # GitHub CLI timeout in milliseconds (default: 1500)
```

#### GitHub Integration (for system tests)
```bash
export UPSTREAM_GH_ACCOUNT="upstream-account"
export UPSTREAM_GH_TOKEN="ghp_xxxxxxxxxxxx"
export FORK_GH_ACCOUNT="your-account"
export FORK_GH_TOKEN="ghp_yyyyyyyyyyyy"
```

#### Jira Integration
```bash
export JIRA_EMAIL="your-email@company.com"
export JIRA_API_TOKEN="your-jira-api-token"
```

## Workflow Examples

### Single Remote Workflow (Direct Repository Access)
For repositories where you have direct push access and don't need a fork:

```bash
# 1. Clone and work directly
git clone https://github.com/your-org/your-repo.git
cd your-repo

# 2. Create feature branch (no fork needed!)
qs dev "feature-awesome-feature"
# qs automatically detects single remote mode

# 3. Make changes and upload
# ... edit files ...
qs u -m "Add awesome feature"

# 4. Create pull request to the same repository
qs pr

# 5. Clean up after merge
qs dev -d
```

**Note**: `qs` automatically detects single remote mode when you don't have an upstream remote and works seamlessly with just the `origin` remote.

### Fork Workflow (Contributing to External Projects)
For contributing to repositories you don't have direct access to:

```bash
# 1. Fork and setup
git clone https://github.com/original/repo.git
cd repo
qs fork

# 2. Create feature branch
qs dev "feature-awesome-feature"

# 3. Make changes and upload
# ... edit files ...
qs u -m "Add awesome feature"

# 4. Create pull request to upstream
qs pr

# 5. Clean up after merge
qs dev -d
```

### Issue-based Development
```bash
# 1. Create branch from GitHub issue
qs dev https://github.com/owner/repo/issues/123

# 2. Work on the issue
# ... make changes ...
qs u -m "Fix issue #123: resolve the problem"

# 3. Create linked PR
qs pr  # Automatically links to issue #123
```

### Jira Ticket Workflow
```bash
# 1. Create branch from Jira ticket
qs dev https://company.atlassian.net/browse/PROJ-456

# 2. Develop feature
# ... implement changes ...
qs u -m "PROJ-456: implement new feature"

# 3. Create PR with Jira link
qs pr  # Includes Jira ticket reference
```

## System Tests

qs includes a comprehensive system test suite that validates functionality with real GitHub repositories:

```bash
# Run system tests (requires GitHub tokens)
go test -v sys_test.go

# Tests cover:
# - Repository forking and configuration
# - Branch creation and management
# - Pull request workflows
# - Upload/download operations
# - GitHub and Jira integration
```

See [design-systests.md](design-systests.md) for detailed system test documentation.

## Troubleshooting

### Common Issues

#### Prerequisites Missing
```bash
# Error: command not found: gawk, jq, etc.
# Solution: Install missing dependencies
# macOS: brew install gawk jq
# Linux: sudo apt install gawk jq
# Windows: Ensure Git\usr\bin is in PATH
```

#### GitHub CLI Authentication
```bash
# Error: GitHub CLI check failed
# Solution: Authenticate with GitHub CLI
gh auth login
```

#### Git Configuration Issues
```bash
# Error: bad config variable 'branch..gh-merge-base'
# Solution: Fix malformed Git config
git config --remove-section 'branch.'
# Or edit .git/config manually to remove malformed entries
```

#### Version Check Failures
```bash
# Skip version checks in CI/CD environments
export QS_SKIP_QS_VERSION_CHECK=true
```

### Debug Mode
```bash
# Enable verbose output for debugging
qs -v <command>

# Example: Debug fork operation
qs -v fork
```

### Network Issues
- All network operations include automatic retry mechanisms
- Configure retry behavior with environment variables
- Check GitHub CLI connectivity: `gh auth status`

## Contributing

### Development Setup
```bash
# Clone and build
git clone https://github.com/untillpro/qs.git
cd qs
go build

# Run tests
go test ./...

# Run system tests (requires GitHub tokens)
go test -v sys_test.go
```

### Code Structure
- `internal/commands/` - Command implementations
- `internal/cmdproc/` - Command processing and CLI setup
- `internal/systrun/` - System test framework
- `gitcmds/` - Git operation wrappers
- `internal/helper/` - Utility functions and retry logic

### Adding New Commands
1. Add command constant to `internal/commands/const.go`
2. Implement command in `internal/commands/`
3. Add command setup in `internal/cmdproc/cmdproc.go`
4. Add system tests in `sys_test.go`

## License

This project is licensed under the terms specified in the repository.

## Support

- 📖 **Documentation**: [design-systests.md](design-systests.md)
- 🐛 **Issues**: [GitHub Issues](https://github.com/untillpro/qs/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/untillpro/qs/discussions)
