package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/daaku/cssdalek/internal/cssselector"
	"github.com/daaku/cssdalek/internal/cssusage"
	"github.com/daaku/cssdalek/internal/pa"

	"github.com/jpillora/opts"
	"github.com/pkg/errors"
	"github.com/tdewolff/parse/v2/css"
	"github.com/tdewolff/parse/v2/html"
)

var (
	idB         = []byte("id")
	classB      = []byte("class")
	atMediaB    = []byte("@media")
	atFontFaceB = []byte("@font-face")
)

type app struct {
	CSSGlobs  []string `opts:"name=css,short=c,help=Globs targeting CSS files"`
	HTMLGlobs []string `opts:"name=html,short=h,help=Globs targeting HTML files"`
	Include   []string `opts:"short=i,help=Selectors to always include"`

	seenNodesMu sync.Mutex
	seenNodes   []cssselector.Selector

	cssInfoMu sync.Mutex
	cssInfo   *cssusage.Info

	log *log.Logger
}

func (a *app) startGlobJobs(glob string, processor func(string) error) error {
	matches, err := filepath.Glob(glob)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, match := range matches {
		if err := processor(match); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) includeSelector(chain []*cssselector.Selector) bool {
	//TODO: fixme explicit Includes
	/*
		for _, other := range a.Include {
			if selector == other {
				return true
			}
		}
	*/
	pending := len(chain)
	found := make([]bool, pending)
	for _, node := range a.seenNodes {
		for i, selector := range chain {
			if found[i] {
				continue
			}
			if selector.Matches(&node) {
				pending--
				if pending == 0 {
					return true
				}

				found[i] = true
			}
		}
	}
	return false
}

func (a *app) htmlFileProcessor(filename string) error {
	a.log.Printf("Processing HTML file: %s\n", filename)
	f, err := os.Open(filename)
	if err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(a.htmlProcessor(bufio.NewReader(f)))
}

func (a *app) htmlProcessor(r io.Reader) error {
	l := html.NewLexer(r)
	var seenNodes []cssselector.Selector
docloop:
	for {
		tt, _ := l.Next()
		switch tt {
		case html.ErrorToken:
			err := l.Err()
			if err == io.EOF {
				break docloop
			}
			return errors.WithMessagef(err, "at offset %d", l.Offset())
		case html.StartTagToken:
			tag := cssselector.Selector{
				Tag: string(l.Text()),
			}
		tagloop:
			for {
				ttAttr, _ := l.Next()
				switch ttAttr {
				default:
					return errors.Errorf("unexpected token type %s at offset %d", ttAttr, l.Offset())
				case html.AttributeToken:
					name := l.Text()
					if bytes.EqualFold(name, idB) {
						tag.ID = string(bytes.Trim(l.AttrVal(), `"'`))
					} else if bytes.EqualFold(name, classB) {
						classes := bytes.Fields(l.AttrVal())
						tag.Class = make(map[string]struct{})
						for _, c := range classes {
							c := bytes.Trim(c, `"'`)
							tag.Class[string(c)] = struct{}{}
						}
					} else {
						if tag.Attr == nil {
							tag.Attr = make(map[string]struct{})
						}
						tag.Attr[string(name)] = struct{}{}
					}
				case html.StartTagCloseToken:
					break tagloop
				}
			}
			seenNodes = append(seenNodes, tag)
		}
	}

	a.seenNodesMu.Lock()
	a.seenNodes = append(a.seenNodes, seenNodes...)
	a.seenNodesMu.Unlock()

	return nil
}

func (a *app) cssFileProcessor(filename string) error {
	a.log.Printf("Processing CSS file: %s\n", filename)
	f, err := os.Open(filename)
	if err != nil {
		return errors.WithStack(err)
	}
	bw := bufio.NewWriter(os.Stdout)
	if err := a.cssProcessor(bufio.NewReader(f), bw); err != nil {
		return err
	}
	return errors.WithStack(bw.Flush())
}

func (a *app) cssFileUsageProcessor(filename string) error {
	a.log.Printf("Extracting Usage from CSS file: %s\n", filename)
	f, err := os.Open(filename)
	if err != nil {
		return errors.WithStack(err)
	}
	return a.cssUsageExtractor(bufio.NewReader(f))
}

func (a *app) cssUsageExtractor(r io.Reader) error {
	info, err := cssusage.Extract(r)
	if err != nil {
		return err
	}

	a.cssInfoMu.Lock()
	if a.cssInfo == nil {
		a.cssInfo = info
	} else {
		a.cssInfo.Merge(info)
	}
	a.cssInfoMu.Unlock()

	return nil
}

type cssProcessor struct {
	app              *app
	parser           *css.Parser
	data             []byte
	out              io.Writer
	scratch          bytes.Buffer
	mediaQueries     [][]byte
	selectorIncluded bool
	inFontFace       bool
}

func (c *cssProcessor) excludeRuleset() pa.Next {
	for {
		gt, _, _ := c.parser.Next()
		if gt == css.EndRulesetGrammar {
			return c.outer
		}
	}
}

func (c *cssProcessor) error() pa.Next {
	err := c.parser.Err()
	if err == io.EOF {
		return nil
	}
	panic(errors.WithStack(err))
}

func (c *cssProcessor) selector() pa.Next {
	c.scratch.Reset()
	for _, val := range c.parser.Values() {
		c.scratch.Write(val.Data)
	}

	selectorBytes := c.scratch.Bytes()
	chain, err := cssselector.Parse(bytes.NewReader(selectorBytes))
	if err != nil {
		panic(err)
	}

	include := c.app.includeSelector(chain)
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
		c.app.log.Printf("Excluding selector: %s\n", c.scratch.String())
	}

	return c.outer
}

func (c *cssProcessor) beginRuleset() pa.Next {
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

func (c *cssProcessor) decl() pa.Next {
	pa.Write(c.out, c.data)
	pa.WriteString(c.out, ":")
	for _, val := range c.parser.Values() {
		pa.Write(c.out, val.Data)
	}
	pa.WriteString(c.out, ";")
	return c.outer
}

func (c *cssProcessor) beginAtMedia() pa.Next {
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

func (c *cssProcessor) beginAtFontFace() pa.Next {
	pa.Write(c.out, c.data)
	for _, val := range c.parser.Values() {
		pa.Write(c.out, val.Data)
	}
	pa.WriteString(c.out, "{")
	c.inFontFace = true
	return c.outer
}

func (c *cssProcessor) beginAtRule() pa.Next {
	if bytes.EqualFold(c.data, atMediaB) {
		return c.beginAtMedia
	}
	if bytes.EqualFold(c.data, atFontFaceB) {
		return c.beginAtFontFace
	}
	panic(fmt.Sprintf("unimplemented: %s", c.data))
}

func (c *cssProcessor) endAtRule() pa.Next {
	if c.inFontFace {
		c.inFontFace = false
		pa.WriteString(c.out, "}")
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

func (c *cssProcessor) outer() pa.Next {
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

func (a *app) cssProcessor(r io.Reader, w io.Writer) error {
	return pa.Finish((&cssProcessor{
		app:    a,
		out:    w,
		parser: css.NewParser(r, false),
	}).outer)
}

func (a *app) run() error {
	// TODO: css and html usage extractors can run concurrently
	for _, glob := range a.HTMLGlobs {
		if err := a.startGlobJobs(glob, a.htmlFileProcessor); err != nil {
			return err
		}
	}
	for _, glob := range a.CSSGlobs {
		if err := a.startGlobJobs(glob, a.cssFileUsageProcessor); err != nil {
			return err
		}
	}
	for _, glob := range a.CSSGlobs {
		if err := a.startGlobJobs(glob, a.cssFileProcessor); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	a := &app{
		log: log.New(os.Stderr, ">> ", 0),
	}
	opts.Parse(a)
	if err := a.run(); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}
