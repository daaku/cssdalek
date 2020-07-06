package htmlusage

import (
	"errors"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/daaku/cssdalek/internal/cssselector"

	"github.com/daaku/ensure"
)

func seen(t testing.TB, selectors ...string) []cssselector.Selector {
	var parsed []cssselector.Selector
	for _, s := range selectors {
		p, err := cssselector.Parse(strings.NewReader(s))
		ensure.Nil(t, err, "for selector", s)
		parsed = append(parsed, p...)
	}
	return parsed
}

func TestInfoMerge(t *testing.T) {
	i1 := Info{Seen: seen(t, "a")}
	i2 := Info{Seen: seen(t, "b")}
	i1.Merge(&i2)
	ensure.DeepEqual(t, i1, Info{
		Seen: []cssselector.Selector{
			{Tag: "a"},
			{Tag: "b"},
		},
	})
}

func TestValid(t *testing.T) {
	cases := []struct {
		name string
		html string
		seen []cssselector.Selector
	}{
		{
			name: "tag",
			html: `<a>`,
			seen: seen(t, "a"),
		},
		{
			name: "tag - lowercased",
			html: `<A>`,
			seen: seen(t, "a"),
		},
		{
			name: "id",
			html: `<a id="f">`,
			seen: seen(t, "a#f"),
		},
		{
			name: "id - lowercased",
			html: `<A ID="F">`,
			seen: seen(t, "a#f"),
		},
		{
			name: "class",
			html: `<a class="f">`,
			seen: seen(t, "a.f"),
		},
		{
			name: "class - lowercased",
			html: `<A CLASS="F">`,
			seen: seen(t, "a.f"),
		},
		{
			name: "attr",
			html: `<a foo="bar">`,
			seen: []cssselector.Selector{
				{
					Tag:  "a",
					Attr: map[string]struct{}{"foo": {}},
				},
			},
		},
		{
			name: "attr - lowercased",
			html: `<A FOO="BAR">`,
			seen: []cssselector.Selector{
				{
					Tag:  "a",
					Attr: map[string]struct{}{"foo": {}},
				},
			},
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			info, err := Extract(strings.NewReader(c.html))
			ensure.Nil(t, err)
			ensure.DeepEqual(t, info.Seen, c.seen)
		})
	}
}

func TestInvalidHTML(t *testing.T) {
	_, err := Extract(strings.NewReader(`<a <!--`))
	ensure.Err(t, err, regexp.MustCompile("unexpected token"))
}

func TestReaderError(t *testing.T) {
	f, err := ioutil.TempFile("", "cssdalek-htmlusage-")
	ensure.Nil(t, err)
	f.Close()
	os.Remove(f.Name())
	_, err = Extract(f)
	ensure.True(t, errors.Is(err, os.ErrClosed))
}
