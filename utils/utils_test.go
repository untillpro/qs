/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeleteDup(t *testing.T) {
	str := deleteDupMinus("13427-Show--must----go---on")
	require.Equal(t, "13427-Show-must-go-on", str)
	str = deleteDupMinus("----Show--must----")
	require.Equal(t, "-Show-must-", str)
}
