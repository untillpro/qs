package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeleteDup(t *testing.T) {
	str := deleteDupMinus("13427-Show--must----go---on")
	assert.Equal(t, "13427-Show-must-go-on", str)
	str = deleteDupMinus("----Show--must----")
	assert.Equal(t, "-Show-must-", str)
}
