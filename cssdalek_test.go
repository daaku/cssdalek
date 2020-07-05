package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daaku/cssdalek/internal/cssselector"
	"github.com/daaku/ensure"
	"github.com/pkg/errors"
)

func TestCore(t *testing.T) {
	filenames, err := filepath.Glob("testdata/*.1")
	ensure.Nil(t, err)
	for _, filename := range filenames {
		filename := filename
		t.Run(filename, func(t *testing.T) {
			contents, err := ioutil.ReadFile(filename)
			ensure.Nil(t, err)
			parts := bytes.SplitN(contents, []byte("\n--\n"), 3)
			ensure.DeepEqual(t, len(parts), 3)
			logger := log.New(ioutil.Discard, "", 0)
			if testing.Verbose() {
				logger = log.New(os.Stdout, "", 0)
			}
			a := app{log: logger}
			ensure.Nil(t, a.htmlProcessor(bytes.NewReader(parts[0])))
			ensure.Nil(t, a.cssUsageExtractor(bytes.NewReader(parts[1])))
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

func TestWriterError(t *testing.T) {
	a := app{
		log:       log.New(ioutil.Discard, "", 0),
		seenNodes: []cssselector.Selector{{Tag: "a"}},
	}
	pr, pw := io.Pipe()
	pr.Close()
	err := a.cssProcessor(strings.NewReader(`a{color:red}`), pw)
	ensure.True(t, errors.Is(err, io.ErrClosedPipe))
}
