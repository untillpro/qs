package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSum(t *testing.T) {
	require.Equal(t, 1+2, Sum(1, 2))
}
