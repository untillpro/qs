# Design of the GitHub actions to run system tests

## Introduction
This document describes the design of the GitHub actions that run system tests of the project. The GitHub actions are designed to run system tests in a controlled environment.

## Motivation
The motivation behind this design is to ensure that system tests are run automatically whenever changes are made to the codebase. This helps in maintaining the quality of the code and catching issues early in the development process.

## Definitions

- **System Tests**: Tests which located in `qs/sys_test.go` file and are designed to test the system as a whole. Each test uses two GitHub accounts: one for the upstream repository and another for the forked repository.
- **GitHub Actions**: A CI/CD tool provided by GitHub that allows you to automate workflows, such as running tests, building applications, and deploying code.
- **Workflow**: A series of steps defined in a YAML file that describes how the system tests should be executed. Each workflow can include multiple jobs.
- **Environment**: The environment in which the job runs. This can include the operating system, dependencies, and other configurations.


## Principles

- Systems test must be run on Windows, Linux, and macOS
- Use secrets to store GitHub tokens and accounts
- Use `qs/design-systests.md` file to learn about the design of the system tests



