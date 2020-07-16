// +build gofuzz

package fuzz

import (
	"bytes"
	"io/ioutil"
	"log"

	"github.com/daaku/cssdalek/internal/csspurge"
	"github.com/daaku/cssdalek/internal/cssusage"
	"github.com/daaku/cssdalek/internal/usage"
)

func Fuzz(b []byte) int {
	l := log.New(ioutil.Discard, "", 0)
	_ = csspurge.Purge(
		usage.MultiInfo{},
		&cssusage.Info{},
		l,
		bytes.NewReader(b),
		ioutil.Discard,
	)
	return 0
}
