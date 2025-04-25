package systrun

import (
	"testing"
)

func NewSystemTest(t *testing.T, cfg SystemTestCfg) *SystemTest {
	return &SystemTest{
		t:   t,
		cfg: cfg,
	}
}
