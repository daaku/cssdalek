package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daaku/ensure"
)

func TestCSSProcessor(t *testing.T) {
	filenames, err := filepath.Glob("testdata/*.1")
	ensure.Nil(t, err)
	for _, filename := range filenames {
		filename := filename
		t.Run(filename, func(t *testing.T) {
			contents, err := ioutil.ReadFile(filename)
			ensure.Nil(t, err)
			parts := bytes.SplitN(contents, []byte("\n--\n"), 3)
			ensure.DeepEqual(t, len(parts), 3)
			var a = app{log: log.New(ioutil.Discard, "", 0)}
			ensure.Nil(t, a.htmlProcessor(bytes.NewReader(parts[0])))
			var actual bytes.Buffer
			ensure.Nil(t, a.cssProcessor(bytes.NewReader(parts[1]), &actual))
			expected := strings.Replace(strings.TrimSpace(string(parts[2])), "\n", "", -1)
			if strings.TrimSpace(actual.String()) != expected {
				ensure.DeepEqual(t,
					strings.TrimSpace(actual.String()),
					expected,
					"seen nodes", a.seenNodes,
				)
			}
		})
	}
}
