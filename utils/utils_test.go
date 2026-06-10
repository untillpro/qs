/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// BenchmarkAppendNoPrealloc demonstrates ineffective array usage:
// every append that exceeds capacity causes a full slice reallocation and copy.
func BenchmarkAppendNoPrealloc(b *testing.B) {
	const n = 10_000
	for b.Loop() {
		s := []int{}
		for i := range n {
			s = append(s, i)
		}
		_ = s
	}
}

// BenchmarkAppendPrealloc shows the effective alternative:
// a single allocation with make avoids all intermediate copies.
func BenchmarkAppendPrealloc(b *testing.B) {
	const n = 10_000
	for b.Loop() {
		s := make([]int, 0, n)
		for i := range n {
			s = append(s, i)
		}
		_ = s
	}
}

func TestDeleteDup(t *testing.T) {
	str := deleteDupMinus("13427-Show--must----go---on")
	require.Equal(t, "13427-Show-must-go-on", str)
	str = deleteDupMinus("----Show--must----")
	require.Equal(t, "-Show-must-", str)
}
