package systrun

import "testing"

type SystemTestCfg struct {
	GithubAccount         string
	GithubToken           string
	GithubOrg             string
	UpstreamGithubAccount string
	UpstreamRepoName      string
}

type SystemTest struct {
	t         *testing.T
	clonePath string
	cfg       SystemTestCfg
	cases     []TestCase
}

type TestCase struct {
	// Name is the name of the test case.
	Name string
	// Stderr is the expected error output of the command.
	Stderr string
	// Stdout is the expected output of the command.
	Stdout string
	// Cmd is the command to run.
	Cmd string
	// Args are the arguments to the command.
	Args []string
	// CheckResults is intented to check the results of the test case.
	CheckResults func(t *testing.T)
	// ErrorExpected is true if the test case is expected to fail, false otherwise.
	ErrorExpected bool
}
