// +build gofuzz

package fuzz

import (
	"bytes"

	"github.com/daaku/cssdalek/internal/wordusage"
)

func Fuzz(b []byte) int {
	_, _ = wordusage.Extract(bytes.NewReader(b))
	return 0
}
