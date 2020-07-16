// +build gofuzz

package fuzz

import (
	"bytes"

	"github.com/daaku/cssdalek/internal/htmlusage"
)

func Fuzz(b []byte) int {
	_, _ = htmlusage.Extract(bytes.NewReader(b))
	return 0
}
