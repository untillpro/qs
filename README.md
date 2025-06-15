# Quick source management tool

## Installation Prerequisites

### gh    version > 2.27
- https://github.com/cli/cli
- MSI: https://github.com/cli/cli/releases/tag/v2.3.0
- chocolatey: choco install ghhttps://github.com/untillpro/qs/blob/main/README.md
  - https://docs.chocolatey.org/en-us/choco/setup#install-with-cmd.exe
  - Also
    - choco install gh
    - choco install git
    - choco install golang

### git

- $Git\usr\bin must be in PATH
  - gives `yes`, `grep`, `sed`, `jq` and other Unix utilities

### xclip
For linux additionally:   sudo apt install xclip

## Installation

go install github.com/untillpro/qs@latest

## Basic Usage
Must be run in cloned github repo folder.
Path qs.exe should be added to system PATH env variable.

- `qs`: show status of current folder

- `qs d`                    : Download sources (git pull)
- `qs u -m [message]`       : Upload sources (git add + commit + push)
- `qs r`                    : Create release
- `qs g`                    : Shows Git GUI
- `qs -h`, `qs --help`      : Help for qs
- `qs -v`, `qs --verbose`   : Verbose output

- `qs fork`  		            : Forks repo to user's account and creates upstream
- `qs dev repo-name`        : Make new dev branch with name repo-name
                              Repo-name can be copied as 
                                - [Name and Permanent link] from Project Kaiser task. 
                                - URL of github issue with issue number.
                              If a buffer contains link on issue number, qs dev creates brnach and links it to github issue

- `qs dev -d`         	    : Deletes branches in user's repository, which were accepted & mergred to parent repo
- `qs dev -i`, `qs dev --ignore-hook`    : Create developer branch and does not ask to add hook against large files.
- `qs dev -n`, `qs dev --no-fork`        : Allows to create developer branch in trunk.
- `qs pr`                   : Create pull request. Title is taken from name of topic in "qs dev" command
                            : If a branch is linked to github issue, qs pr makes a pull request, linked to that issue.
- `qs pr -d`, qs pr --draft : Create pull request draft.
- `qs pr merge [PR URL]`    : Merge pull request.

- `qs upgrade`  	          : Shows command to install qs
- `qs version`  	          : Shows version of currently installed qs

## Usage with JIRA
Create branch from Jira issue use "qs dev https://untill.atlassian.net/browse/Issue-Number"
Example:
```
1. qs dev https://untill.atlassian.net/browse/AIR-270
```
In order it works define the following env variables:
- JIRA_EMAIL=[user email]
- JIRA_API_TOKEN=[user jira api token] - it's genertated [here]([url](https://id.atlassian.com/manage-profile/security/api-tokens)).

If JIRA_EMAIL is not defined qs tries to get it from local git settings.
If it's not defined, qs gives error "Error: Please export JIRA_EMAIL."
If JIRA_API_TOKEN is not found, qs gives error:

```
--------------------------------------------------------------------------------
Error: JIRA API token not found. Please set environment variable JIRA_API_TOKEN.
            Jira API token can generate on this page:
          https://id.atlassian.com/manage-profile/security/api-tokens           
--------------------------------------------------------------------------------
```

2. "qs pr" creates Pull Request with link on Jira issue.
  
> **Note**
  - `qs u` takes comment from clipboard. If current branch is "main/master", 
           and message is empty or very short (<3 symbols), qs willask to enter message.
           If the message is too short, it shows error:   
                  ----  Too short comment not allowed! --- 

## Prevent large commits

Command 'qs dev' creates a developer branch and after success, it shows the following question:

```
   Git pre-commit hook, preventing commit large files does not exist.
   Do you want to set hook(y/n)?
```

- On 'y', qs creates github local pre-commit hook script for current repository.

If local pre-commit hook found, 'qs dev' does not asks to create the hook.

## SSH Configuration

In order yo work system tests we need to configure SSH keys for GitHub.
We need 2 GitHub accounts: one for the "origin" (fork) and one for the "upstream" (main) repository. 

Example of SSH config:
```
# For the "origin" GitHub account
Host github-origin
    HostName github.com
    User git
    IdentityFile ~/.ssh/privatekey_for_fork_account
    IdentitiesOnly yes

# For the "upstream" GitHub account
Host github-upstream
    HostName github.com
    User git
    IdentityFile ~/.ssh/privatekey_for_upstream_account
    IdentitiesOnly yes

```

