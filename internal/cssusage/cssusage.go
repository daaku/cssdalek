// Package cssusage extracts usage information from CSS. This includes things
// like font-face and keyframes.
package cssusage

import (
	"bytes"
	"io"
	"strings"

	"github.com/daaku/cssdalek/internal/cssselector"
	"github.com/daaku/cssdalek/internal/pa"

	"github.com/pkg/errors"
	"github.com/tdewolff/parse/v2/css"
)

var (
	fontFamilyB = []byte("font-family")
	commaB      = []byte(",")
	quotesS     = `"'`
)

type extractor struct {
	parser           *css.Parser
	data             []byte
	currentSelectors []string
	currentFontFaces []string
	scratch          bytes.Buffer

	// map of font faces to selectors that use them
	fontFace map[string][]cssselector.Chain
}

type Info struct {
	FontFace map[string][]cssselector.Chain
}

func (i *Info) Merge(other *Info) {
	if i.FontFace == nil {
		i.FontFace = make(map[string][]cssselector.Chain)
	}
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
	for _, fontFace := range c.currentFontFaces {
		// collect font faces if any
		if c.fontFace == nil {
			c.fontFace = make(map[string][]cssselector.Chain)
		}
		selectors, found := c.fontFace[fontFace]
		if !found {
			selectors = make([]cssselector.Chain, 0, len(c.currentSelectors))
		}
		for _, selector := range c.currentSelectors {
			chain, err := cssselector.Parse(strings.NewReader(selector))
			if err != nil {
				panic(err)
			}
			selectors = append(selectors, chain)
		}
		c.fontFace[fontFace] = selectors
	}

	// reset everything
	c.currentSelectors = c.currentSelectors[:0]
	c.currentFontFaces = c.currentFontFaces[:0]

	return c.outer
}

func (c *extractor) scrUnqStr() string {
	s := string(bytes.Trim(c.scratch.Bytes(), quotesS))
	c.scratch.Reset()
	return s
}

func (c *extractor) decl() pa.Next {
	// decl without selector means we're inside an @ rule
	if len(c.currentSelectors) == 0 {
		return c.outer
	}

	if bytes.EqualFold(c.data, fontFamilyB) {
		c.scratch.Reset()
		for _, val := range c.parser.Values() {
			if bytes.Equal(val.Data, commaB) {
				if c.scratch.Len() != 0 {
					c.currentFontFaces = append(c.currentFontFaces, c.scrUnqStr())
				}
				continue
			}
			c.scratch.Write(val.Data)
		}
		if c.scratch.Len() != 0 {
			c.currentFontFaces = append(c.currentFontFaces, c.scrUnqStr())
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
