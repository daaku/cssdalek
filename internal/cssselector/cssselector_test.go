package cssselector

import (
	"fmt"
	"strings"
	"testing"

	"github.com/daaku/ensure"
)

func set(values ...string) map[string]struct{} {
	s := make(map[string]struct{})
	for _, v := range values {
		s[v] = struct{}{}
	}
	return s
}

func TestIsZeroTrue(t *testing.T) {
	ensure.True(t, (&Selector{}).IsZero())

	var s *Selector
	ensure.True(t, s.IsZero())
}

func TestIsZeroFalse(t *testing.T) {
	cases := []*Selector{
		{Tag: "a"},
		{ID: "a"},
		{Class: set("a")},
		{Attr: set("a")},
	}
	for _, c := range cases {
		c := c
		name := fmt.Sprintf("%+v", c)
		t.Run(name, func(t *testing.T) {
			ensure.False(t, c.IsZero())
		})
	}
}

func TestSelectorMatchTrue(t *testing.T) {
	cases := []struct {
		name     string
		selector Selector
		node     Selector
	}{
		{
			"just a tag",
			Selector{
				Tag: "a",
			},
			Selector{
				Tag: "a",
			},
		},
		{
			"tag and other crap",
			Selector{
				Tag: "a",
			},
			Selector{
				Tag: "a",
				ID:  "b",
			},
		},
		{
			"just a id",
			Selector{
				ID: "a",
			},
			Selector{
				ID: "a",
			},
		},
		{
			"id and other crap",
			Selector{
				ID: "a",
			},
			Selector{
				ID:  "a",
				Tag: "b",
			},
		},
		{
			"just a class",
			Selector{
				Class: set("a"),
			},
			Selector{
				Class: set("a"),
			},
		},
		{
			"class and other crap",
			Selector{
				Class: set("a"),
			},
			Selector{
				Class: set("a"),
				ID:    "b",
			},
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			ensure.True(t, c.selector.Matches(&c.node))
		})
	}
}

func TestSelectorMatchFalse(t *testing.T) {
	cases := []struct {
		name     string
		selector Selector
		node     Selector
	}{
		{
			"tag",
			Selector{
				Tag: "a",
			},
			Selector{
				Tag: "b",
			},
		},
		{
			"id",
			Selector{
				ID: "a",
			},
			Selector{
				ID: "b",
			},
		},
		{
			"class",
			Selector{
				Class: set("a"),
			},
			Selector{
				Class: set("b"),
			},
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			ensure.False(t, c.selector.Matches(&c.node))
		})
	}
}

func TestValidSelectors(t *testing.T) {
	cases := []struct {
		name     string
		text     string
		selector []*Selector
	}{
		{
			"hash",
			"#first-id",
			[]*Selector{
				{ID: "first-id"},
			},
		},
		{
			"descendant hash",
			"#first-id #second-id",
			[]*Selector{
				{ID: "first-id"},
				{ID: "second-id"},
			},
		},
		{
			"class",
			".first-class",
			[]*Selector{
				{Class: set("first-class")},
			},
		},
		{
			"descendant class",
			".first-class .second-class",
			[]*Selector{
				{Class: set("first-class")},
				{Class: set("second-class")},
			},
		},
		{
			"and class",
			".first-class.second-class",
			[]*Selector{
				{Class: set("first-class", "second-class")},
			},
		},
		{
			"direct descandant",
			".first-class > .second-class",
			[]*Selector{
				{Class: set("first-class")},
				{Class: set("second-class")},
			},
		},
		{
			"preceed",
			".first-class ~ .second-class",
			[]*Selector{
				{Class: set("first-class")},
				{Class: set("second-class")},
			},
		},
		{
			"immediately preceed",
			".first-class + .second-class",
			[]*Selector{
				{Class: set("first-class")},
				{Class: set("second-class")},
			},
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			actual, err := Parse(strings.NewReader(c.text))
			ensure.Nil(t, err)
			ensure.DeepEqual(t, actual, c.selector)
		})
	}
}
