// +build gofuzz

package fuzz

import (
	"bytes"

	"github.com/daaku/cssdalek/internal/cssusage"
)

func Fuzz(b []byte) int {
	_, _ = cssusage.Extract(bytes.NewReader(b))
	return 0
}
