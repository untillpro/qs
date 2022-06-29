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
	str, _ := getBranchName("Show", "must", "go", "on", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, str, "13427-Show-must-go-on")
	str, _ = getBranchName("Show   ivv?", "must    ", "go", "on---", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, str, "13427-Show-ivv-must-go-on")
	str, _ = getBranchName("Show", "must", "go", "on")
	assert.Equal(t, str, "Show-must-go-on")
	str, _ = getBranchName("Show")
	assert.Equal(t, str, "Show")
	str, _ = getBranchName("Show   ivv? must $   go on---  https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, str, "13427-Show-ivv-must-go-on")
	str, _ = getBranchName("Show   ivv? must $   ", "go on---  https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, str, "13427-Show-ivv-must-go-on")
	str, _ = getBranchName("Show   ivv? must $   go  on---", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, str, "13427-Show-ivv-must-go-on")
	str, _ = getBranchName("Show", "ivv? must $   go  on--- https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, str, "13427-Show-ivv-must-go-on")
	str, _ = getBranchName("Show", "ivv? must $   go  on---", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, str, "13427-Show-ivv-must-go-on")
	str, _ = getBranchName("q", "dev", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, str, "13427-q-dev")

	//Logn name
	str, _ = getBranchName("Show", "me this  very long string more than fifty symbols in lenth with long task number 11111111111111", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, str, "13427-Show-me-this-very-long-string-more-than-fift")
}
