package wordusage

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/daaku/cssdalek/internal/cssselector"
	"github.com/daaku/ensure"
)

func set(values ...string) map[string]struct{} {
	s := make(map[string]struct{})
	for _, v := range values {
		s[v] = struct{}{}
	}
	return s
}

func TestInfoMerge(t *testing.T) {
	i1 := Info{}
	i2 := Info{Seen: set("a", "b")}
	i3 := Info{Seen: set("b", "c")}
	i1.Merge(&i2)
	i1.Merge(&i3)
	ensure.DeepEqual(t, i1, Info{Seen: set("a", "b", "c")})
}

func TestIncludes(t *testing.T) {
	cases := []struct {
		name     string
		seen     map[string]struct{}
		selector string
	}{
		{
			name:     "tag",
			seen:     set("a"),
			selector: "a",
		},
		{
			name:     "id",
			seen:     set("foo"),
			selector: "#foo",
		},
		{
			name:     "class",
			seen:     set("foo"),
			selector: ".foo",
		},
		{
			name:     "attr",
			seen:     set("foo"),
			selector: "[foo]",
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			chain, err := cssselector.Parse(strings.NewReader(c.selector))
			ensure.Nil(t, err)
			i := Info{Seen: c.seen}
			ensure.True(t, i.Includes(chain))
		})
	}
}

func TestDoesNotInclude(t *testing.T) {
	cases := []struct {
		name     string
		seen     map[string]struct{}
		selector string
	}{
		{
			name:     "tag",
			seen:     set("a"),
			selector: "strong",
		},
		{
			name:     "id",
			seen:     set("foo"),
			selector: "#bar",
		},
		{
			name:     "class",
			seen:     set("foo"),
			selector: ".bar",
		},
		{
			name:     "attr",
			seen:     set("foo"),
			selector: "[bar]",
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			chain, err := cssselector.Parse(strings.NewReader(c.selector))
			ensure.Nil(t, err)
			i := Info{Seen: c.seen}
			ensure.False(t, i.Includes(chain))
		})
	}
}

func TestExtract(t *testing.T) {
	cases := []struct {
		name string
		in   string
		seen map[string]struct{}
	}{
		{
			name: "tag",
			in:   `<a>`,
			seen: set("a"),
		},
		{
			name: "id",
			in:   `<a id="foo">`,
			seen: set("a", "id", "foo"),
		},
		{
			name: "class",
			in:   `<a class="foo">`,
			seen: set("a", "class", "foo"),
		},
		{
			name: "class with dashes",
			in:   `<a class="foo-bar">`,
			seen: set("a", "class", "foo-bar"),
		},
		{
			name: "a word",
			in:   `foo`,
			seen: set("foo"),
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			info, err := Extract(strings.NewReader(c.in))
			ensure.Nil(t, err)
			ensure.DeepEqual(t, info.Seen, c.seen)
		})
	}
}

func TestReaderError(t *testing.T) {
	f, err := ioutil.TempFile("", "cssdalek-wordusage-")
	ensure.Nil(t, err)
	f.Close()
	os.Remove(f.Name())
	_, err = Extract(f)
	ensure.True(t, errors.Is(err, os.ErrClosed))
}
