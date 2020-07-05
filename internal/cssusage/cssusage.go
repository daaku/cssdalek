// Package cssusage extracts usage information from CSS. This includes things
// like font-face and keyframes.
package cssusage

import (
	"bytes"
	"io"

	"github.com/daaku/cssdalek/internal/pa"

	"github.com/pkg/errors"
	"github.com/tdewolff/parse/v2/css"
)

var (
	fontFamilyB = []byte("font-family")
)

type extractor struct {
	parser           *css.Parser
	data             []byte
	currentSelectors []string
	currentFontFaces []string
	scratch          bytes.Buffer

	// map of font faces to selectors that use them
	fontFace map[string][]string
}

type Info struct {
	FontFace map[string][]string
}

func (i *Info) Merge(other *Info) {
	for face, selectors := range other.FontFace {
		i.FontFace[face] = append(i.FontFace[face], selectors...)
	}
}

func Extract(r io.Reader) (*Info, error) {
	e := &extractor{
		parser: css.NewParser(r, false),
	}
	if err := pa.Finish(e.outer); err != nil {
		return nil, err
	}
	return &Info{
		FontFace: e.fontFace,
	}, nil
}

func (c *extractor) error() pa.Next {
	err := c.parser.Err()
	if err == io.EOF {
		return nil
	}
	panic(errors.WithStack(err))
}

func (c *extractor) selector() pa.Next {
	c.scratch.Reset()
	for _, val := range c.parser.Values() {
		c.scratch.Write(val.Data)
	}
	c.currentSelectors = append(c.currentSelectors, c.scratch.String())
	return c.outer
}

func (c *extractor) endRuleset() pa.Next {
	for _, selector := range c.currentSelectors {
		// collect font faces if any
		if c.fontFace == nil {
			c.fontFace = make(map[string][]string)
		}
		faces, found := c.fontFace[selector]
		if !found {
			faces = make([]string, 0, len(c.currentFontFaces))
		}
		c.fontFace[selector] = append(faces, c.currentFontFaces...)
	}

	// reset everything
	c.currentSelectors = c.currentSelectors[:0]
	c.currentFontFaces = c.currentFontFaces[:0]

	return c.outer
}

func (c *extractor) decl() pa.Next {
	// decl without selector means we're inside @font-face
	if len(c.currentSelectors) == 0 {
		return c.outer
	}

	if bytes.EqualFold(c.data, fontFamilyB) {
		for _, val := range c.parser.Values() {
			c.currentFontFaces = append(c.currentFontFaces, string(bytes.Trim(val.Data, `"'`)))
		}
	}
	return c.outer
}

func (c *extractor) outer() pa.Next {
	gt, _, data := c.parser.Next()
	c.data = data
	switch gt {
	default:
		return c.outer
	case css.ErrorGrammar:
		return c.error
	case css.QualifiedRuleGrammar:
		return c.selector
	case css.BeginRulesetGrammar:
		return c.selector
	case css.EndRulesetGrammar:
		return c.endRuleset
	case css.DeclarationGrammar:
		return c.decl
	}
}
