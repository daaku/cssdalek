// +build gofuzz

package fuzz

import (
	"bytes"

	"github.com/daaku/cssdalek/internal/cssselector"
)

func Fuzz(b []byte) int {
	_, _ = cssselector.Parse(bytes.NewReader(b))
	return 0
}
