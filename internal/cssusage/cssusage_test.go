package cssusage

import (
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/daaku/cssdalek/internal/cssselector"

	"github.com/daaku/ensure"
	"github.com/pkg/errors"
)

func TestInfoMerge(t *testing.T) {
	c1, err := cssselector.Parse(strings.NewReader("a"))
	ensure.Nil(t, err)
	c2, err := cssselector.Parse(strings.NewReader("i"))
	ensure.Nil(t, err)
	c3, err := cssselector.Parse(strings.NewReader("b"))
	ensure.Nil(t, err)

	i0 := Info{}
	i1 := Info{
		FontFace: map[string][]cssselector.Chain{
			"f1": {c1, c2},
			"f2": {c1, c3},
		},
	}
	i2 := Info{
		FontFace: map[string][]cssselector.Chain{
			"f1": {c2, c3},
			"f3": {c1, c2},
		},
	}
	i0.Merge(&i1)
	i0.Merge(&i2)
	ensure.DeepEqual(t, i0, Info{
		FontFace: map[string][]cssselector.Chain{
			"f1": {c1, c2, c2, c3},
			"f2": {c1, c3},
			"f3": {c1, c2},
		},
	})
}

func TestFontFace(t *testing.T) {
	aC, err := cssselector.Parse(strings.NewReader("a"))
	ensure.Nil(t, err)
	aiC, err := cssselector.Parse(strings.NewReader("a i"))
	ensure.Nil(t, err)

	cases := []struct {
		name  string
		css   string
		faces map[string][]cssselector.Chain
		kf    map[string][]cssselector.Chain
	}{
		{
			name: "unquoted",
			css:  `a { font-family: Sans; }`,
			faces: map[string][]cssselector.Chain{
				"Sans": {aC},
			},
		},
		{
			name: "quoted",
			css:  `a { font-family: "Sans"; }`,
			faces: map[string][]cssselector.Chain{
				"Sans": {aC},
			},
		},
		{
			name: "invalid unquoted",
			css:  `a { font-family: Sans Serif; }`,
			faces: map[string][]cssselector.Chain{
				"Sans Serif": {aC},
			},
		},
		{
			name: "nested tags",
			css:  `a i { font-family: Sans; }`,
			faces: map[string][]cssselector.Chain{
				"Sans": {aiC},
			},
		},
		{
			name: "multiple families",
			css:  `a { font-family: Sans, Serif; }`,
			faces: map[string][]cssselector.Chain{
				"Sans":  {aC},
				"Serif": {aC},
			},
		},
		{
			name: "multiple families quoted",
			css:  `a { font-family: "Sans", "Serif"; }`,
			faces: map[string][]cssselector.Chain{
				"Sans":  {aC},
				"Serif": {aC},
			},
		},
		{
			name: "multiple families invalid unquoted",
			css:  `a { font-family: Sans Serif, Comic Sans; }`,
			faces: map[string][]cssselector.Chain{
				"Sans Serif": {aC},
				"Comic Sans": {aC},
			},
		},
		{
			name: "multiple families invalid comma",
			css:  `a { font-family: Sans, , Serif; }`,
			faces: map[string][]cssselector.Chain{
				"Sans":  {aC},
				"Serif": {aC},
			},
		},
		{
			name: "multiple selectors",
			css:  `a, a i { font-family: Sans; }`,
			faces: map[string][]cssselector.Chain{
				"Sans": {aC, aiC},
			},
		},
		{
			name: "font-face at-rule is ignored",
			css:  `@font-face { font-family: Foo; }`,
		},
		{
			name: "keyframe at-rule is ignored",
			css:  `@keyframes { 0% {} }`,
		},
		{
			name: "keyframes in animation",
			css:  `a { animation: foo; }`,
			kf: map[string][]cssselector.Chain{
				"foo": {aC},
			},
		},
		{
			name: "keyframes in animation-name",
			css:  `a { animation-name: foo bar "baz jax"; }`,
			kf: map[string][]cssselector.Chain{
				"foo":     {aC},
				"bar":     {aC},
				"baz jax": {aC},
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			info, err := Extract(strings.NewReader(c.css))
			ensure.Nil(t, err)
			ensure.DeepEqual(t, info.FontFace, c.faces, "faces")
			ensure.DeepEqual(t, info.Keyframes, c.kf, "keyframes")
		})
	}
}

func TestReaderError(t *testing.T) {
	f, err := ioutil.TempFile("", "cssdalek-cssusage-")
	ensure.Nil(t, err)
	f.Close()
	os.Remove(f.Name())
	_, err = Extract(f)
	ensure.True(t, errors.Is(err, os.ErrClosed))
}

func TestInvalidSelector(t *testing.T) {
	const css = `a # { font-family: Sans; }`
	_, err := Extract(strings.NewReader(css))
	ensure.Err(t, err, regexp.MustCompile("unexpected token"))
}
