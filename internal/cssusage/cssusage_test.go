package cssusage

import (
	"strings"
	"testing"

	"github.com/daaku/ensure"
)

func TestFontFace(t *testing.T) {
	cases := []struct {
		name  string
		css   string
		faces map[string][]string
	}{
		{
			name:  "unquoted",
			css:   `a { font-family: Sans; }`,
			faces: map[string][]string{"a": {"Sans"}},
		},
		{
			name:  "quoted",
			css:   `a { font-family: "Sans"; }`,
			faces: map[string][]string{"a": {"Sans"}},
		},
		{
			name:  "invalid unquoted",
			css:   `a { font-family: Sans Serif; }`,
			faces: map[string][]string{"a": {"Sans Serif"}},
		},
		{
			name:  "nested tags",
			css:   `a i { font-family: Sans; }`,
			faces: map[string][]string{"a i": {"Sans"}},
		},
		{
			name:  "multiple families",
			css:   `a { font-family: Sans, Serif; }`,
			faces: map[string][]string{"a": {"Sans", "Serif"}},
		},
		{
			name:  "multiple families quoted",
			css:   `a { font-family: "Sans", "Serif"; }`,
			faces: map[string][]string{"a": {"Sans", "Serif"}},
		},
		{
			name:  "multiple families invalid unquoted",
			css:   `a { font-family: Sans Serif, Comic Sans; }`,
			faces: map[string][]string{"a": {"Sans Serif", "Comic Sans"}},
		},
		{
			name:  "multiple families invalid comma",
			css:   `a { font-family: Sans, , Serif; }`,
			faces: map[string][]string{"a": {"Sans", "Serif"}},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			info, err := Extract(strings.NewReader(c.css))
			ensure.Nil(t, err)
			ensure.DeepEqual(t, info.FontFace, c.faces)
		})
	}
}
