package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/daaku/cssdalek/internal/csspurge"
	"github.com/daaku/cssdalek/internal/cssusage"
	"github.com/daaku/cssdalek/internal/htmlusage"

	"github.com/facebookgo/errgroup"
	"github.com/jpillora/opts"
	"github.com/pkg/errors"
)

type app struct {
	CSSGlobs  []string `opts:"name=css,short=c,help=globs targeting CSS files"`
	HTMLGlobs []string `opts:"name=html,short=h,help=globs targeting HTML files"`
	Include   []string `opts:"short=i,help=selectors to always include"`

	htmlInfoMu sync.Mutex
	htmlInfo   htmlusage.Info

	cssInfoMu sync.Mutex
	cssInfo   cssusage.Info

	log *log.Logger
}

func (a *app) buildHTMLInfo(eg *errgroup.Group) {
	defer eg.Done()
	eg.Add(len(a.HTMLGlobs))
	for _, glob := range a.HTMLGlobs {
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
					a.log.Printf("Processing HTML file: %s\n", filename)
					f, err := os.Open(filename)
					if err != nil {
						eg.Error(errors.WithStack(err))
						return
					}
					info, err := htmlusage.Extract(bufio.NewReader(f))
					if err != nil {
						eg.Error(err)
						return
					}
					a.htmlInfoMu.Lock()
					a.htmlInfo.Merge(info)
					a.htmlInfoMu.Unlock()
				}()
			}
		}()
	}
}

func (a *app) buildCSSInfo(eg *errgroup.Group) {
	defer eg.Done()
	eg.Add(len(a.CSSGlobs))
	for _, glob := range a.CSSGlobs {
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
					a.log.Printf("Processing CSS file: %s\n", filename)
					f, err := os.Open(filename)
					if err != nil {
						eg.Error(errors.WithStack(err))
						return
					}
					info, err := cssusage.Extract(bufio.NewReader(f))
					if err != nil {
						eg.Error(err)
						return
					}
					a.cssInfoMu.Lock()
					a.cssInfo.Merge(info)
					a.cssInfoMu.Unlock()
				}()
			}
		}()
	}
}

func (a *app) run() error {
	start := time.Now()
	var eg errgroup.Group
	eg.Add(2)
	go a.buildHTMLInfo(&eg)
	go a.buildCSSInfo(&eg)
	if err := eg.Wait(); err != nil {
		return err
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
			err = csspurge.Purge(&a.htmlInfo, &a.cssInfo, a.log, bufio.NewReader(f), w)
			if err != nil {
				return err
			}
		}
	}
	a.log.Println("Took", time.Since(start))
	return errors.WithStack(w.Flush())
}

func main() {
	a := app{log: log.New(os.Stderr, ">> ", 0)}
	opts.Parse(&a)
	if err := a.run(); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}
