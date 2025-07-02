# Design of the GitHub actions to run system tests

## Introduction
This document describes the design of the GitHub actions that run system tests of the project. The GitHub actions are designed to run system tests in a controlled environment.

## Motivation
The motivation behind this design is to ensure that system tests are run automatically whenever changes are made to the codebase. This helps in maintaining the quality of the code and catching issues early in the development process.

## Definitions

- **System Tests**: Tests which located in `sys_test.go` file and are designed to test the system as a whole. Each test uses two GitHub accounts: one for the upstream repository and another for the forked repository.
- **GitHub Actions**: A CI/CD tool provided by GitHub that allows you to automate workflows, such as running tests, building applications, and deploying code.
- **Workflow**: A series of steps defined in a YAML file that describes how the system tests should be executed. Each workflow can include multiple jobs.
- **Environment**: The environment in which the job runs. This can include the operating system, dependencies, and other configurations.


## Principles

- Systems test must be run on Windows, Linux, and macOS
- Use secrets to store GitHub tokens and accounts
- Use `design-systests.md` file to learn about the design of the system tests

## Implementation

### Workflow File
The workflow file must be located in `.github/workflows/sys_tests.yml`. 

### Secrets

Workflow requires the following environment variables to be set in secrets:
- UPSTREAM_GH_ACCOUNT: The GitHub account used for the upstream repository.
- UPSTREAM_GH_TOKEN: The GitHub token for the upstream account.
- FORK_GH_ACCOUNT: The GitHub account used for the forked repository.
- FORK_GH_TOKEN: The GitHub token for the forked account.
- JIRA_EMAIL: The email address associated with the JIRA account.
- JIRA_API_TOKEN: The API token for the JIRA account.
- JIRA_TICKET_URL: The URL of the JIRA ticket used in the system tests.



