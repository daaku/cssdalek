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
	"github.com/jpillora/opts"
	"github.com/pkg/errors"
	"github.com/tdewolff/parse/v2/css"
	"github.com/tdewolff/parse/v2/html"
)

type app struct {
	CSSGlobs  []string `opts:"name=css,short=c,help=Globs targeting CSS files"`
	HTMLGlobs []string `opts:"name=html,short=h,help=Globs targeting HTML files"`
	Include   []string `opts:"short=i,help=Selectors to always include"`

	seenNodesMu sync.Mutex
	seenNodes   []cssselector.Selector

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

var (
	idB    = []byte("id")
	classB = []byte("class")
)

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

var atMedia = []byte("@media")

func (a *app) cssProcessor(r io.Reader, w io.Writer) error {
	p := css.NewParser(r, false)

	var scratch bytes.Buffer
	selectorIncluded := false

	var mediaQueries [][]byte

	processSelector := func() error {
		scratch.Reset()
		for _, val := range p.Values() {
			scratch.Write(val.Data)
		}

		selectorBytes := scratch.Bytes()
		chain, err := cssselector.Parse(bytes.NewReader(selectorBytes))
		if err != nil {
			return err
		}

		include := a.includeSelector(chain)
		if include {
			// included, and we need to write a comma since we already wrote one
			if selectorIncluded {
				if _, err := io.WriteString(w, ","); err != nil {
					return errors.WithStack(err)
				}
			}
			selectorIncluded = true

			// write all pending media queries, if any since we're including something
			// contained within
			for _, mq := range mediaQueries {
				if _, err := w.Write(mq); err != nil {
					return errors.WithStack(err)
				}
				if _, err := io.WriteString(w, "{"); err != nil {
					return errors.WithStack(err)
				}
			}
			mediaQueries = mediaQueries[:0]

			// now write the selector itself
			if _, err := w.Write(selectorBytes); err != nil {
				return errors.WithStack(err)
			}
		} else {
			a.log.Printf("Excluding selector: %s\n", scratch.String())
		}

		return nil
	}

	excluding := false
	//TODO: AtRules need special handling?
outer:
	for {
		gt, _, data := p.Next()

		// if skipping, keep skipping until the end
		if excluding {
			if gt == css.EndRulesetGrammar {
				excluding = false
			}
			continue outer
		}

		switch gt {
		default:
			if _, err := w.Write(data); err != nil {
				return errors.WithStack(err)
			}
		case css.ErrorGrammar:
			err := p.Err()
			if err == io.EOF {
				break outer
			}
			return errors.WithStack(err)
		case css.QualifiedRuleGrammar:
			if err := processSelector(); err != nil {
				return err
			}
		case css.BeginRulesetGrammar:
			if err := processSelector(); err != nil {
				return err
			}

			// if we haven't included any so far, we're excluding the entire ruleset
			if !selectorIncluded {
				excluding = true
				continue outer
			}

			// otherwise we just began the ruleset
			if _, err := io.WriteString(w, "{"); err != nil {
				return errors.WithStack(err)
			}

			// reset
			selectorIncluded = false
		case css.DeclarationGrammar:
			if _, err := w.Write(data); err != nil {
				return errors.WithStack(err)
			}
			if _, err := io.WriteString(w, ":"); err != nil {
				return errors.WithStack(err)
			}
			for _, val := range p.Values() {
				if _, err := w.Write(val.Data); err != nil {
					return errors.WithStack(err)
				}
			}
			if _, err := io.WriteString(w, ";"); err != nil {
				return errors.WithStack(err)
			}
		case css.CommentGrammar:
			continue outer
		case css.AtRuleGrammar, css.BeginAtRuleGrammar:
			if bytes.EqualFold(data, atMedia) {
				scratch.Reset()
				scratch.Write(data)
				for _, val := range p.Values() {
					scratch.Write(val.Data)
				}
				query := make([]byte, scratch.Len())
				copy(query, scratch.Bytes())
				mediaQueries = append(mediaQueries, query)
				continue outer
			}
			panic(fmt.Sprintf("unimplemented: %s", data))
		case css.EndAtRuleGrammar:
			if len(mediaQueries) > 0 {
				// we did not write this media query, so throw it away and don't write a
				// closing }
				mediaQueries = mediaQueries[:len(mediaQueries)-1]
			} else {
				// we already wrote the query, so write a corresponding close
				if _, err := io.WriteString(w, "}"); err != nil {
					return errors.WithStack(err)
				}
			}
		}
	}
	return nil
}

func (a *app) run() error {
	for _, glob := range a.HTMLGlobs {
		if err := a.startGlobJobs(glob, a.htmlFileProcessor); err != nil {
			return err
		}
	}
	fmt.Printf("%+v\n", a.seenNodes)
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
