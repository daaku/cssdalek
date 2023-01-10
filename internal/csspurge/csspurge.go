// Package csspurge purges unused CSS.
package csspurge

import (
	"bytes"
	"io"
	"log"

	"github.com/daaku/cssdalek/internal/cssselector"
	"github.com/daaku/cssdalek/internal/cssusage"
	"github.com/daaku/cssdalek/internal/pa"
	"github.com/daaku/cssdalek/internal/usage"

	"github.com/pkg/errors"
	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

var (
	atMediaB          = []byte("@media")
	atSupportsB       = []byte("@supports")
	atFontFaceB       = []byte("@font-face")
	atKeyframes       = []byte("@keyframes")
	atWebkitKeyframes = []byte("@-webkit-keyframes")
	fontFamilyB       = []byte("font-family")
	licenseCommentB   = []byte("/*!")
	sourceMapCommentB = []byte("/*#")
	quotesS           = `"'`
)

func Purge(u usage.Info, c *cssusage.Info, l *log.Logger, r io.Reader, w io.Writer) error {
	p := purger{
		usageInfo: u,
		cssInfo:   c,
		log:       l,
		parser:    css.NewParser(parse.NewInput(r), false),
		out:       w,
	}
	return pa.Finish(p.outer)
}

type purger struct {
	usageInfo        usage.Info
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
	inKeyframes      bool
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
	include := true
	if !c.inKeyframes {
		chain, err := cssselector.Parse(bytes.NewReader(selectorBytes))
		if err != nil {
			panic(errors.WithMessagef(err, "at offset %d", c.parser.Offset()))
		}
		include = c.usageInfo.Includes(chain)
	}

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

func (c *purger) comment() pa.Next {
	if bytes.HasPrefix(c.data, licenseCommentB) || bytes.HasPrefix(c.data, sourceMapCommentB) {
		pa.Write(c.out, c.data)
		pa.WriteString(c.out, "\n")
	}
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

func (c *purger) beginAtKeyframes() pa.Next {
	c.scratch.Reset()
	for _, val := range c.parser.Values() {
		c.scratch.Write(val.Data)
	}
	keyframesName := bytes.TrimSpace(c.scratch.Bytes())

	if selectors, found := c.cssInfo.Keyframes[string(keyframesName)]; found {
		for _, s := range selectors {
			if c.usageInfo.Includes(s) {
				c.inKeyframes = true
				return c.beginAtRuleUnknown
			}
		}
	}

	return c.dropUntilEndAtRule
}

func (c *purger) dropUntilEndAtRule() pa.Next {
	for tt, _, _ := c.parser.Next(); tt != css.EndAtRuleGrammar; tt, _, _ = c.parser.Next() {
	}
	return c.outer
}

func (c *purger) atRule() pa.Next {
	pa.Write(c.out, c.data)
	for _, val := range c.parser.Values() {
		pa.Write(c.out, val.Data)
	}
	pa.WriteString(c.out, ";")
	return c.outer
}

func (c *purger) beginAtRuleUnknown() pa.Next {
	pa.Write(c.out, c.data)
	for _, val := range c.parser.Values() {
		pa.Write(c.out, val.Data)
	}
	pa.WriteString(c.out, "{")
	return c.outer
}

func (c *purger) beginAtRule() pa.Next {
	if bytes.EqualFold(c.data, atMediaB) || bytes.EqualFold(c.data, atSupportsB) {
		return c.beginAtMedia
	}
	if bytes.EqualFold(c.data, atFontFaceB) {
		return c.beginAtFontFace
	}
	if bytes.EqualFold(c.data, atKeyframes) || bytes.EqualFold(c.data, atWebkitKeyframes) {
		return c.beginAtKeyframes
	}
	return c.beginAtRuleUnknown
}

func (c *purger) endAtRule() pa.Next {
	c.inKeyframes = false

	if c.inFontFace {
		pa.WriteString(c.out, "}")

		if selectors, found := c.cssInfo.FontFace[c.fontFaceName]; found {
			for _, s := range selectors {
				if c.usageInfo.Includes(s) {
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
	case css.DeclarationGrammar, css.CustomPropertyGrammar:
		return c.decl
	case css.CommentGrammar:
		return c.comment
	case css.AtRuleGrammar:
		return c.atRule
	case css.BeginAtRuleGrammar:
		return c.beginAtRule
	case css.EndAtRuleGrammar:
		return c.endAtRule
	}
}
