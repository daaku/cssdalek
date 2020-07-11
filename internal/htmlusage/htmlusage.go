// Package htmlusage extracts HTML usage information.
package htmlusage

import (
	"bytes"
	"io"
	"strings"

	"github.com/daaku/cssdalek/internal/cssselector"

	"github.com/pkg/errors"
	"github.com/tdewolff/parse/v2/html"
)

var (
	idB    = []byte("id")
	classB = []byte("class")
)

type Info struct {
	Seen []cssselector.Selector
}

func (i *Info) Merge(other *Info) {
	i.Seen = append(i.Seen, other.Seen...)
}

func (i *Info) Includes(chain cssselector.Chain) bool {
	pending := len(chain)
	found := make([]bool, pending)
	for _, node := range i.Seen {
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

func FromSelectors(ss []string) (*Info, error) {
	var i Info
	for _, s := range ss {
		sel, err := cssselector.Parse(strings.NewReader(s))
		if err != nil {
			return nil, errors.WithMessagef(err, "invalid selector: %q", s)
		}
		i.Seen = append(i.Seen, sel...)
	}
	return &i, nil
}

func Extract(r io.Reader) (*Info, error) {
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
			return nil, errors.WithMessagef(err, "at offset %d", l.Offset())
		case html.StartTagToken:
			tag := cssselector.Selector{
				Tag: string(bytes.ToLower(l.Text())),
			}
		tagloop:
			for {
				ttAttr, _ := l.Next()
				switch ttAttr {
				default:
					return nil, errors.Errorf("unexpected token type %s at offset %d", ttAttr, l.Offset())
				case html.AttributeToken:
					name := l.Text()
					if bytes.EqualFold(name, idB) {
						tag.ID = string(bytes.ToLower(bytes.Trim(l.AttrVal(), `"'`)))
					} else if bytes.EqualFold(name, classB) {
						classes := bytes.Fields(l.AttrVal())
						tag.Class = make(map[string]struct{})
						for _, c := range classes {
							c := bytes.ToLower(bytes.Trim(c, `"'`))
							tag.Class[string(c)] = struct{}{}
						}
					} else {
						if tag.Attr == nil {
							tag.Attr = make(map[string]struct{})
						}
						tag.Attr[string(bytes.ToLower(name))] = struct{}{}
					}
				case html.StartTagCloseToken:
					break tagloop
				}
			}
			seenNodes = append(seenNodes, tag)
		}
	}

	return &Info{
		Seen: seenNodes,
	}, nil
}
