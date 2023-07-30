package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTaskIDFromUrl(t *testing.T) {
	reponame := retrieveRepoNameFromUPL("https://github.com/IVVORG/test-repo/pull/38")
	assert.Equal(t, reponame, "IVVORG/test-repo")
}

func TestGetNotes(t *testing.T) {
	// Test for PK types
	s1 := "Permanent support for Peter,  Pascal, and customers "
	s2 := " https://dev.untill.com/projects/#!361164  "
	notes := []string{s1, s2}
	title, url := GetNoteAndURL(notes)
	assert.Equal(t, title, "Permanent support for Peter,  Pascal, and customers")
	assert.Equal(t, url, "https://dev.untill.com/projects/#!361164")

	notes = []string{s2, s1}
	title, url = GetNoteAndURL(notes)
	assert.Equal(t, title, "Permanent support for Peter,  Pascal, and customers")
	assert.Equal(t, url, "https://dev.untill.com/projects/#!361164")

	// Test for GH Issue types

	s1 = "Resolves issue #324 My Best problem ever"
	s2 = "Resolves #324"
	notes = []string{s1, s2}
	title, url = GetNoteAndURL(notes)
	assert.Equal(t, title, "Resolves issue #324 My Best problem ever")
	assert.Equal(t, url, "")
}

func TestGetBody(t *testing.T) {
	// Test for PK types
	s1 := "Permanent support for Peter, Pascal, and customers "
	s2 := " https://dev.untill.com/projects/#!361164  "
	notes := []string{s1, s2}
	body := getBodyFromNotes(notes)
	assert.Equal(t, body, "")

	// Test for GH Issue types
	s1 = "Resolves issue #324 My Best problem ever"
	s2 = " Resolves #324  "
	notes = []string{s1, s2}
	body = getBodyFromNotes(notes)
	assert.Equal(t, body, "Resolves #324")
}
