package csspurge

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daaku/cssdalek/internal/cssusage"
	"github.com/daaku/cssdalek/internal/htmlusage"
	"github.com/tdewolff/minify/v2/css"

	"github.com/daaku/ensure"
)

func minify(t *testing.T, b []byte) []byte {
	var out bytes.Buffer
	err := css.Minify(nil, &out, bytes.NewReader(b), nil)
	ensure.Nil(t, err)
	return out.Bytes()
}

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
			htmlInfo, err := htmlusage.Extract(bytes.NewReader(parts[0]))
			ensure.Nil(t, err)
			cssInfo, err := cssusage.Extract(bytes.NewReader(parts[1]))
			ensure.Nil(t, err)
			var actualB bytes.Buffer
			ensure.Nil(t, Purge(htmlInfo, cssInfo, logger, bytes.NewReader(parts[1]), &actualB))
			expected := string(minify(t, parts[2]))
			actual := string(minify(t, actualB.Bytes()))
			if expected != actual {
				ensure.DeepEqual(t,
					strings.TrimSpace(actualB.String()),
					strings.TrimSpace(string(parts[2])),
					"html info", htmlInfo,
					"css info", cssInfo,
				)
			}
		})
	}
}

func TestReaderError(t *testing.T) {
	f, err := ioutil.TempFile("", "cssdalek-csspurge-")
	ensure.Nil(t, err)
	f.Close()
	os.Remove(f.Name())
	err = Purge(nil, nil, nil, f, ioutil.Discard)
	ensure.True(t, errors.Is(err, os.ErrClosed))
}
