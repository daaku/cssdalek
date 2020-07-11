package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/daaku/cssdalek/internal/csspurge"
	"github.com/daaku/cssdalek/internal/cssusage"
	"github.com/daaku/cssdalek/internal/htmlusage"
	"github.com/daaku/cssdalek/internal/includeusage"
	"github.com/daaku/cssdalek/internal/usage"
	"github.com/daaku/cssdalek/internal/wordusage"

	"github.com/facebookgo/errgroup"
	"github.com/jpillora/opts"
	"github.com/pkg/errors"
)

func buildRe(ss []string) ([]*regexp.Regexp, error) {
	var res = make([]*regexp.Regexp, len(ss))
	var err error
	for i, s := range ss {
		res[i], err = regexp.Compile(s)
		if err != nil {
			return nil, errors.WithMessagef(err, "invalid regexp: %q", s)
		}
	}
	return res, nil
}

type app struct {
	CSSGlobs        []string `opts:"name=css,short=c,help=globs targeting CSS files"`
	HTMLGlobs       []string `opts:"name=html,short=h,help=globs targeting HTML files"`
	WordGlobs       []string `opts:"name=word,short=w,help=globs targeting word files"`
	IncludeClass    []string `opts:"help=class regexp to include"`
	IncludeID       []string `opts:"help=id regexp to include"`
	IncludeSelector []string `opts:"short=i,help=selectors to include"`
	Verbose         bool     `opts:"short=v,help=verbose logging"`

	htmlInfoMu sync.Mutex
	htmlInfo   htmlusage.Info

	wordInfoMu sync.Mutex
	wordInfo   wordusage.Info

	cssInfoMu sync.Mutex
	cssInfo   cssusage.Info

	log *log.Logger
}

func (a *app) build(eg *errgroup.Group, globs []string, b func(r io.Reader) error) {
	defer eg.Done()
	eg.Add(len(globs))
	for _, glob := range globs {
		glob := glob
		go func() {
			defer eg.Done()
			matches, err := filepath.Glob(glob)
			if err != nil {
				eg.Error(errors.WithStack(err))
				return
			}
			eg.Add(len(matches))
			for _, filename := range matches {
				filename := filename
				go func() {
					defer eg.Done()
					a.log.Printf("Processing file: %s\n", filename)
					f, err := os.Open(filename)
					if err != nil {
						eg.Error(errors.WithStack(err))
						return
					}
					defer f.Close()
					if err := b(bufio.NewReader(f)); err != nil {
						eg.Error(errors.WithMessagef(err, "in file: %q", filename))
						return
					}
				}()
			}
		}()
	}

}

func (a *app) buildHTMLInfo(r io.Reader) error {
	info, err := htmlusage.Extract(r)
	if err != nil {
		return err
	}
	a.htmlInfoMu.Lock()
	a.htmlInfo.Merge(info)
	a.htmlInfoMu.Unlock()
	return nil
}

func (a *app) buildWordInfo(r io.Reader) error {
	info, err := wordusage.Extract(r)
	if err != nil {
		return err
	}
	a.wordInfoMu.Lock()
	a.wordInfo.Merge(info)
	a.wordInfoMu.Unlock()
	return nil
}

func (a *app) buildCSSInfo(r io.Reader) error {
	info, err := cssusage.Extract(r)
	if err != nil {
		return err
	}
	a.cssInfoMu.Lock()
	a.cssInfo.Merge(info)
	a.cssInfoMu.Unlock()
	return nil
}

func (a *app) run() error {
	start := time.Now()

	if a.Verbose {
		a.log = log.New(os.Stderr, ">> ", 0)
	} else {
		a.log = log.New(ioutil.Discard, "", 0)
	}

	includeClass, err := buildRe(a.IncludeClass)
	if err != nil {
		return err
	}

	includeID, err := buildRe(a.IncludeID)
	if err != nil {
		return err
	}

	includeSelector, err := htmlusage.FromSelectors(a.IncludeSelector)
	if err != nil {
		return err
	}

	var eg errgroup.Group
	eg.Add(3)
	go a.build(&eg, a.HTMLGlobs, a.buildHTMLInfo)
	go a.build(&eg, a.WordGlobs, a.buildWordInfo)
	go a.build(&eg, a.CSSGlobs, a.buildCSSInfo)
	if err := eg.Wait(); err != nil {
		return err
	}

	usageInfo := usage.MultiInfo{
		&includeusage.IncludeClass{Re: includeClass},
		&includeusage.IncludeID{Re: includeID},
		includeSelector,
		&a.htmlInfo,
		&a.wordInfo,
	}

	w := bufio.NewWriter(os.Stdout)
	for _, glob := range a.CSSGlobs {
		matches, err := filepath.Glob(glob)
		if err != nil {
			return errors.WithStack(err)
		}
		for _, filename := range matches {
			f, err := os.Open(filename)
			if err != nil {
				return errors.WithStack(err)
			}
			err = csspurge.Purge(usageInfo, &a.cssInfo, a.log, bufio.NewReader(f), w)
			if err != nil {
				return errors.WithMessagef(err, "in file %q", filename)
			}
		}
	}
	a.log.Println("Took", time.Since(start))
	return errors.WithStack(w.Flush())
}

func main() {
	var a app
	opts.Parse(&a)
	if err := a.run(); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}
