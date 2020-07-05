package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/daaku/cssdalek/internal/csspurge"
	"github.com/daaku/cssdalek/internal/cssusage"
	"github.com/daaku/cssdalek/internal/htmlusage"

	"github.com/jpillora/opts"
	"github.com/pkg/errors"
)

type app struct {
	CSSGlobs  []string `opts:"name=css,short=c,help=Globs targeting CSS files"`
	HTMLGlobs []string `opts:"name=html,short=h,help=Globs targeting HTML files"`
	Include   []string `opts:"short=i,help=Selectors to always include"`

	htmlInfoMu sync.Mutex
	htmlInfo   *htmlusage.Info

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

func (a *app) htmlFileProcessor(filename string) error {
	a.log.Printf("Processing HTML file: %s\n", filename)
	f, err := os.Open(filename)
	if err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(a.htmlProcessor(bufio.NewReader(f)))
}

func (a *app) htmlProcessor(r io.Reader) error {
	info, err := htmlusage.Extract(r)
	if err != nil {
		return err
	}

	a.htmlInfoMu.Lock()
	if a.htmlInfo == nil {
		a.htmlInfo = info
	} else {
		a.htmlInfo.Merge(info)
	}
	a.htmlInfoMu.Unlock()

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

func (a *app) cssProcessor(r io.Reader, w io.Writer) error {
	return csspurge.Purge(a.htmlInfo, a.cssInfo, a.log, r, w)
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
