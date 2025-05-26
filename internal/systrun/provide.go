package systrun

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// New creates a new SystemTest instance
func New(t *testing.T, testConfig *TestConfig) *SystemTest {
	t.Parallel()

	timestamp := time.Now().Format("060102150405") // YYMMDDhhmmss
	repoName := fmt.Sprintf("%s-%s", testConfig.TestID, timestamp)

	return &SystemTest{
		t:             t,
		cfg:           testConfig,
		repoName:      repoName,
		cloneRepoPath: filepath.Join(TestDataDir, repoName),
	}
}
