package includeusage

import (
	"regexp"
	"strings"
	"testing"

	"github.com/daaku/cssdalek/internal/cssselector"
	"github.com/daaku/ensure"
)

func TestIncludeClass(t *testing.T) {
	cases := []struct {
		name string
		re   []*regexp.Regexp
		s    string
	}{
		{
			name: "first selector",
			re:   []*regexp.Regexp{regexp.MustCompile("^f")},
			s:    ".foo",
		},
		{
			name: "second selector",
			re: []*regexp.Regexp{
				regexp.MustCompile("^a"),
				regexp.MustCompile("f"),
			},
			s: ".foo",
		},
		{
			name: "multiple elements",
			re: []*regexp.Regexp{
				regexp.MustCompile("a"),
				regexp.MustCompile("b"),
				regexp.MustCompile("f"),
			},
			s: ".a .b .foo",
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			i := IncludeClass{Re: c.re}
			s, err := cssselector.Parse(strings.NewReader(c.s))
			ensure.Nil(t, err)
			ensure.True(t, i.Includes(s))
		})
	}
}

func TestNotIncludeClass(t *testing.T) {
	cases := []struct {
		name string
		re   []*regexp.Regexp
		s    string
	}{
		{
			name: "none match",
			re:   []*regexp.Regexp{regexp.MustCompile("^f")},
			s:    ".bar",
		},
		{
			name: "multiple elements",
			re: []*regexp.Regexp{
				regexp.MustCompile("a"),
				regexp.MustCompile("b"),
			},
			s: ".a .b .foo",
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			i := IncludeClass{Re: c.re}
			s, err := cssselector.Parse(strings.NewReader(c.s))
			ensure.Nil(t, err)
			ensure.False(t, i.Includes(s))
		})
	}
}

func TestIncludeID(t *testing.T) {
	cases := []struct {
		name string
		re   []*regexp.Regexp
		s    string
	}{
		{
			name: "first selector",
			re:   []*regexp.Regexp{regexp.MustCompile("^f")},
			s:    "#foo",
		},
		{
			name: "second selector",
			re: []*regexp.Regexp{
				regexp.MustCompile("^a"),
				regexp.MustCompile("f"),
			},
			s: "#foo",
		},
		{
			name: "multiple elements",
			re: []*regexp.Regexp{
				regexp.MustCompile("a"),
				regexp.MustCompile("b"),
				regexp.MustCompile("f"),
			},
			s: "#a #b #foo",
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			i := IncludeID{Re: c.re}
			s, err := cssselector.Parse(strings.NewReader(c.s))
			ensure.Nil(t, err)
			ensure.True(t, i.Includes(s))
		})
	}
}

func TestNotIncludeID(t *testing.T) {
	cases := []struct {
		name string
		re   []*regexp.Regexp
		s    string
	}{
		{
			name: "none match",
			re:   []*regexp.Regexp{regexp.MustCompile("^f")},
			s:    "#bar",
		},
		{
			name: "multiple elements",
			re: []*regexp.Regexp{
				regexp.MustCompile("a"),
				regexp.MustCompile("b"),
			},
			s: "#a #b #foo",
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			i := IncludeID{Re: c.re}
			s, err := cssselector.Parse(strings.NewReader(c.s))
			ensure.Nil(t, err)
			ensure.False(t, i.Includes(s))
		})
	}
}
