package systrun

import (
	"testing"
)

// New creates a new SystemTest instance
func New(t *testing.T, testConfig *TestConfig) *SystemTest {
	return &SystemTest{
		t:             t,
		cfg:           testConfig,
		cloneRepoPath: generateCloneRepoPath(),
	}
}
