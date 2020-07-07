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
	animationB  = []byte("animation")
	commaB      = []byte(",")
	quotesS     = `"'`
)

type extractor struct {
	parser           *css.Parser
	data             []byte
	currentSelectors []string
	currentFontFaces []string
	currentKeyframes []string
	scratch          bytes.Buffer

	info *Info
}

type Info struct {
	FontFace  map[string][]cssselector.Chain
	Keyframes map[string][]cssselector.Chain
}

func (i *Info) Merge(other *Info) {
	if len(other.FontFace) > 0 && i.FontFace == nil {
		i.FontFace = make(map[string][]cssselector.Chain)
	}
	for face, selectors := range other.FontFace {
		i.FontFace[face] = append(i.FontFace[face], selectors...)
	}

	if len(other.Keyframes) > 0 && i.Keyframes == nil {
		i.Keyframes = make(map[string][]cssselector.Chain)
	}
	for kf, selectors := range other.Keyframes {
		i.Keyframes[kf] = append(i.Keyframes[kf], selectors...)
	}
}

func Extract(r io.Reader) (*Info, error) {
	i := &Info{}
	e := &extractor{
		parser: css.NewParser(r, false),
		info:   i,
	}
	if err := pa.Finish(e.outer); err != nil {
		return nil, err
	}
	return i, nil
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
	if len(c.currentFontFaces) == 0 && len(c.currentKeyframes) == 0 {
		return c.outer
	}

	currentSelectors := make([]cssselector.Chain, 0, len(c.currentSelectors))
	for _, selector := range c.currentSelectors {
		chain, err := cssselector.Parse(strings.NewReader(selector))
		if err != nil {
			panic(err)
		}
		currentSelectors = append(currentSelectors, chain)
	}

	for _, fontFace := range c.currentFontFaces {
		if c.info.FontFace == nil {
			c.info.FontFace = make(map[string][]cssselector.Chain)
		}
		c.info.FontFace[fontFace] = append(c.info.FontFace[fontFace], currentSelectors...)
	}

	for _, kf := range c.currentKeyframes {
		if c.info.Keyframes == nil {
			c.info.Keyframes = make(map[string][]cssselector.Chain)
		}
		c.info.Keyframes[kf] = append(c.info.Keyframes[kf], currentSelectors...)
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

	// TODO: support font shorthand
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

	// consider all space separated idents to be keyframe names
	if bytes.EqualFold(c.data, animationB) {
		c.scratch.Reset()
		for _, val := range c.parser.Values() {
			if c.scratch.Len() != 0 {
				c.currentKeyframes = append(c.currentKeyframes, c.scrUnqStr())
			}
			c.scratch.Write(val.Data)
		}
		if c.scratch.Len() != 0 {
			c.currentKeyframes = append(c.currentKeyframes, c.scrUnqStr())
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
