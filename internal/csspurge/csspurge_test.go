package csspurge

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daaku/cssdalek/internal/cssusage"
	"github.com/daaku/cssdalek/internal/htmlusage"
	"github.com/daaku/ensure"
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
			htmlInfo, err := htmlusage.Extract(bytes.NewReader(parts[0]))
			ensure.Nil(t, err)
			cssInfo, err := cssusage.Extract(bytes.NewReader(parts[1]))
			ensure.Nil(t, err)
			var actual bytes.Buffer
			ensure.Nil(t, Purge(htmlInfo, cssInfo, logger, bytes.NewReader(parts[1]), &actual))
			expected := strings.Replace(strings.TrimSpace(string(parts[2])), "\n", "", -1)
			if strings.TrimSpace(actual.String()) != expected {
				ensure.DeepEqual(t,
					strings.TrimSpace(actual.String()),
					expected,
					"html info", htmlInfo,
					"css info", cssInfo,
				)
			}
		})
	}
}
