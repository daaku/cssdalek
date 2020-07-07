// Package csspurge purges unused CSS.
package csspurge

import (
	"bytes"
	"io"
	"log"

	"github.com/daaku/cssdalek/internal/cssselector"
	"github.com/daaku/cssdalek/internal/cssusage"
	"github.com/daaku/cssdalek/internal/htmlusage"
	"github.com/daaku/cssdalek/internal/pa"

	"github.com/pkg/errors"
	"github.com/tdewolff/parse/v2/css"
)

var (
	atMediaB    = []byte("@media")
	atFontFaceB = []byte("@font-face")
	fontFamilyB = []byte("font-family")
	quotesS     = `"'`
)

func Purge(h *htmlusage.Info, c *cssusage.Info, l *log.Logger, r io.Reader, w io.Writer) error {
	p := purger{
		htmlInfo: h,
		cssInfo:  c,
		log:      l,
		parser:   css.NewParser(r, false),
		out:      w,
	}
	return pa.Finish(p.outer)
}

type purger struct {
	htmlInfo         *htmlusage.Info
	cssInfo          *cssusage.Info
	log              *log.Logger
	parser           *css.Parser
	data             []byte
	out              io.Writer
	outSwap          io.Writer
	scratch          bytes.Buffer
	mediaQueries     [][]byte
	selectorIncluded bool
	inFontFace       bool
	fontFaceRule     bytes.Buffer
	fontFaceName     string
}

func (c *purger) excludeRuleset() pa.Next {
	for {
		gt, _, _ := c.parser.Next()
		if gt == css.EndRulesetGrammar {
			return c.outer
		}
	}
}

func (c *purger) error() pa.Next {
	err := c.parser.Err()
	if err == io.EOF {
		return nil
	}
	panic(errors.WithStack(err))
}

func (c *purger) selector() pa.Next {
	c.scratch.Reset()
	for _, val := range c.parser.Values() {
		c.scratch.Write(val.Data)
	}

	selectorBytes := c.scratch.Bytes()
	chain, err := cssselector.Parse(bytes.NewReader(selectorBytes))
	if err != nil {
		panic(err)
	}

	include := c.htmlInfo.Includes(chain)
	if include {
		// write all pending media queries, if any since we're including something
		// contained within
		for _, mq := range c.mediaQueries {
			pa.Write(c.out, mq)
			pa.WriteString(c.out, "{")
		}
		c.mediaQueries = c.mediaQueries[:0]

		// included, and we need to write a comma since we already wrote one
		if c.selectorIncluded {
			pa.WriteString(c.out, ",")
		}
		c.selectorIncluded = true

		// now write the selector itself
		pa.Write(c.out, selectorBytes)
	} else {
		c.log.Printf("Excluding selector: %s\n", c.scratch.String())
	}

	return c.outer
}

func (c *purger) beginRuleset() pa.Next {
	_ = c.selector()

	// if we haven't included any so far, we're excluding the entire ruleset
	if !c.selectorIncluded {
		return c.excludeRuleset
	}

	// otherwise we just began the ruleset
	pa.WriteString(c.out, "{")

	// reset
	c.selectorIncluded = false
	return c.outer
}

func (c *purger) decl() pa.Next {
	if c.inFontFace {
		if bytes.EqualFold(c.data, fontFamilyB) {
			c.scratch.Reset()
			for _, val := range c.parser.Values() {
				c.scratch.Write(val.Data)
			}
			c.fontFaceName = string(bytes.Trim(c.scratch.Bytes(), quotesS))
		}
	}

	pa.Write(c.out, c.data)
	pa.WriteString(c.out, ":")
	for _, val := range c.parser.Values() {
		pa.Write(c.out, val.Data)
	}
	pa.WriteString(c.out, ";")
	return c.outer
}

func (c *purger) beginAtMedia() pa.Next {
	c.scratch.Reset()
	c.scratch.Write(c.data)
	for _, val := range c.parser.Values() {
		c.scratch.Write(val.Data)
	}
	query := make([]byte, c.scratch.Len())
	copy(query, c.scratch.Bytes())
	c.mediaQueries = append(c.mediaQueries, query)
	return c.outer
}

func (c *purger) beginAtFontFace() pa.Next {
	c.inFontFace = true
	c.outSwap = c.out
	c.out = &c.fontFaceRule
	pa.Write(c.out, c.data)
	pa.WriteString(c.out, "{")
	return c.outer
}

func (c *purger) beginAtRuleUnknown() pa.Next {
	pa.Write(c.out, c.data)
	for _, val := range c.parser.Values() {
		pa.Write(c.out, val.Data)
	}
	pa.WriteString(c.out, ";")
	return c.outer
}

func (c *purger) beginAtRule() pa.Next {
	if bytes.EqualFold(c.data, atMediaB) {
		return c.beginAtMedia
	}
	if bytes.EqualFold(c.data, atFontFaceB) {
		return c.beginAtFontFace
	}
	return c.beginAtRuleUnknown
}

func (c *purger) endAtRule() pa.Next {
	if c.inFontFace {
		pa.WriteString(c.out, "}")

		if selectors, found := c.cssInfo.FontFace[c.fontFaceName]; found {
			for _, s := range selectors {
				if c.htmlInfo.Includes(s) {
					io.Copy(c.outSwap, &c.fontFaceRule)
					break
				}
			}
		}

		c.inFontFace = false
		c.fontFaceName = ""
		c.fontFaceRule.Reset()
		c.out = c.outSwap

		return c.outer
	}

	if len(c.mediaQueries) > 0 {
		// we did not write this media query, so throw it away and don't write a
		// closing }
		c.mediaQueries = c.mediaQueries[:len(c.mediaQueries)-1]
	} else {
		// we already wrote the query, so write a corresponding close
		pa.WriteString(c.out, "}")
	}
	return c.outer
}

func (c *purger) outer() pa.Next {
	gt, _, data := c.parser.Next()
	c.data = data
	switch gt {
	default:
		pa.Write(c.out, data)
		return c.outer
	case css.ErrorGrammar:
		return c.error
	case css.QualifiedRuleGrammar:
		return c.selector
	case css.BeginRulesetGrammar:
		return c.beginRuleset
	case css.DeclarationGrammar:
		return c.decl
	case css.CommentGrammar:
		return c.outer
	case css.AtRuleGrammar, css.BeginAtRuleGrammar:
		return c.beginAtRule
	case css.EndAtRuleGrammar:
		return c.endAtRule
	}
}
