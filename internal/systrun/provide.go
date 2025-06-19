package systrun

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/untillpro/qs/internal/cmdproc"
)

// New creates a new SystemTest instance
func New(t *testing.T, testConfig *TestConfig) *SystemTest {
	timestamp := time.Now().Format("060102150405") // YYMMDDhhmmss
	repoName := fmt.Sprintf("%s-%s", testConfig.TestID, timestamp)
	t.Logf("Repo name: %s", repoName)

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return &SystemTest{
		ctx:           context.Background(),
		cfg:           testConfig,
		repoName:      repoName,
		cloneRepoPath: filepath.Join(wd, TestDataDir, repoName),
		qsExecRootCmd: cmdproc.ExecRootCmd,
	}
}
