// Package cssselector parses selectors for the purpose of purging unused CSS.
// It discards various bits of information not used by the purging process, and
// isn't generically useful.
package cssselector

import (
	"bytes"
	"io"

	"github.com/pkg/errors"
	"github.com/tdewolff/parse/v2/css"
)

// Selector is a single parsed selector, a number of which form a chain together.
type Selector struct {
	Tag           string
	ID            string
	Class         map[string]struct{}
	Attr          map[string]struct{}
	PsuedoClass   string
	PsuedoElement string
}

// IsZero returns true of this selector is a zero value. This is also true for
// '*', the universal selector.
func (s *Selector) IsZero() bool {
	return s == nil ||
		(s.Tag == "" &&
			s.ID == "" &&
			len(s.Class) == 0 &&
			len(s.Attr) == 0 &&
			s.PsuedoClass == "" &&
			s.PsuedoElement == "")
}

// Matches returns true of this selector matches the given node.
func (s *Selector) Matches(node *Selector) bool {
	if s.Tag != "" && s.Tag != node.Tag {
		return false
	}
	if s.ID != "" && s.ID != node.ID {
		return false
	}
	for class := range s.Class {
		if _, found := node.Class[class]; !found {
			return false
		}
	}
	for attr := range s.Attr {
		if _, found := node.Attr[attr]; !found {
			return false
		}
	}
	return true
}

type Chain []Selector

func Parse(selector io.Reader) (Chain, error) {
	l := css.NewLexer(selector)
	s := Selector{}
	chain := make(Chain, 0, 1)
outer:
	for {
		tt, data := l.Next()
		switch tt {
		default:
			return nil, errors.Errorf(
				"cssselector: unexpected token %s with data %q at offset %d",
				tt, data, l.Offset())
		case css.ErrorToken:
			err := l.Err()
			if err == io.EOF {
				if !s.IsZero() {
					chain = append(chain, s)
				}
				break outer
			}
			return nil, errors.WithStack(err)
		case css.HashToken:
			s.ID = string(bytes.ToLower(data[1:])) // drop leading #
		case css.ColonToken:
			tt, data := l.Next()
			switch tt {
			default:
				return nil, errors.Errorf(
					"cssselector: unexpected token %s with data %q at offset %d after colon",
					tt, data, l.Offset())
			case css.ColonToken:
				tt, data := l.Next()
				switch tt {
				default:
					return nil, errors.Errorf(
						"cssselector: unexpected token %s with data %q at offset %d after colon",
						tt, data, l.Offset())
				case css.IdentToken:
					s.PsuedoElement = string(bytes.ToLower(data))
				}
			case css.IdentToken:
				s.PsuedoClass = string(bytes.ToLower(data))
			}
		case css.DelimToken:
			if len(data) != 1 {
				return nil, errors.Errorf(
					"cssselector: unexpected token %s with data %q at offset %d while parsing delimiter",
					tt, data, l.Offset())
			}
			switch data[0] {
			default:
				return nil, errors.Errorf(
					"cssselector: unexpected token %s with data %q at offset %d while parsing delimiter",
					tt, data, l.Offset())
			case '.':
				tt, next := l.Next()
				if tt != css.IdentToken {
					return nil, errors.Errorf(
						"cssselector: unexpected token %s with %q followed by %q at offset %d while parsing class selector",
						tt, data, next, l.Offset())
				}
				if s.Class == nil {
					s.Class = make(map[string]struct{})
				}
				s.Class[string(bytes.ToLower(next))] = struct{}{}
			case '>', '+', '~':
				if !s.IsZero() {
					chain = append(chain, s)
					s = Selector{}
				}
			}
		case css.IdentToken:
			s.Tag = string(bytes.ToLower(data))
		case css.WhitespaceToken:
			if !s.IsZero() {
				chain = append(chain, s)
				s = Selector{}
			}
		}
	}
	return chain, nil
}
