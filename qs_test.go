package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeleteDup(t *testing.T) {
	str := deleteDupMinus("13427-Show--must----go---on")
	assert.Equal(t, str, "13427-Show-must-go-on")
	str = deleteDupMinus("----Show--must----")
	assert.Equal(t, str, "-Show-must-")
}
func TestGetTaskIDFromUrl(t *testing.T) {
	topicid := getTaskIDFromURL("https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, topicid, "13427")
	topicid = getTaskIDFromURL("https://dev.heeus.io/launchpad/13428")
	assert.Equal(t, topicid, "13428")
	topicid = getTaskIDFromURL("13429")
	assert.Equal(t, topicid, "13429")
}

func TestGetBranchName(t *testing.T) {
	str := getBranchName("Show", "must", "go", "on", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, str, "13427-Show-must-go-on")
	str = getBranchName("Show   ivv?", "must    ", "go", "on---", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, str, "13427-Show-ivv-must-go-on")
	str = getBranchName("Show", "must", "go", "on")
	assert.Equal(t, str, "Show-must-go-on")
	str = getBranchName("Show")
	assert.Equal(t, str, "Show")
}
