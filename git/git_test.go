package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMainBranch(t *testing.T) {
	str := getMainBranch()
	assert.Equal(t, str, "master")
}
