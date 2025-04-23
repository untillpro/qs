package systrun

import (
	"path/filepath"
	"testing"
)

func NewSystemTest(t *testing.T, cfg SystemTestCfg) ISystemTest {
	clonePath := filepath.Join(testDataDir, cfg.UpstreamRepoName)
	return &SystemTest{
		t:            t,
		cfg:          cfg,
		cloneDirPath: clonePath,
	}
}
