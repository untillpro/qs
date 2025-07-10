package gitcmds

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTaskIDFromUrl(t *testing.T) {
	reponame := retrieveRepoNameFromUPL("https://github.com/IVVORG/test-repo/pull/38")
	assert.Equal(t, "IVVORG/test-repo", reponame)
}

func TestGetNotes(t *testing.T) {
	// Test for PK types
	s1 := "Permanent support for Peter,  Pascal, and customers "
	s2 := " https://dev.untill.com/projects/#!361164  "
	notes := []string{s1, s2}
	title, url := GetNoteAndURL(notes)
	assert.Equal(t, "Permanent support for Peter,  Pascal, and customers", title)
	assert.Equal(t, "https://dev.untill.com/projects/#!361164", url)

	notes = []string{s2, s1}
	title, url = GetNoteAndURL(notes)
	assert.Equal(t, "Permanent support for Peter,  Pascal, and customers", title)
	assert.Equal(t, "https://dev.untill.com/projects/#!361164", url)

	// Test for GH Issue types
	s1 = "Resolves issue #324 My Best problem ever"
	s2 = "Resolves #324"
	notes = []string{s1, s2}
	title, url = GetNoteAndURL(notes)
	assert.Equal(t, "Resolves issue #324 My Best problem ever", title)
	assert.Equal(t, "", url)
}

func TestGetBody(t *testing.T) {
	// Test for PK types
	s1 := "Permanent support for Peter, Pascal, and customers "
	s2 := " https://dev.untill.com/projects/#!361164  "
	notes := []string{s1, s2}
	body := GetBodyFromNotes(notes)
	assert.Equal(t, "", body)

	// Test for GH Issue types
	s1 = "Resolves issue #324 My Best problem ever"
	s2 = " Resolves #324  "
	notes = []string{s1, s2}
	body = GetBodyFromNotes(notes)
	assert.Equal(t, "Resolves #324", body)
}
