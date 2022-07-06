package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTaskIDFromUrl(t *testing.T) {
	reponame := retrieveRepoNameFromUPL("https://github.com/IVVORG/test-repo/pull/38")
	assert.Equal(t, reponame, "IVVORG/test-repo")
}
