package wordusage

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/daaku/cssdalek/internal/cssselector"
	"github.com/pkg/errors"
)

type Info struct {
	Seen map[string]struct{}
}

func (i *Info) Merge(other *Info) {
	if len(other.Seen) > 0 && i.Seen == nil {
		i.Seen = make(map[string]struct{})
	}
	for k := range other.Seen {
		i.Seen[k] = struct{}{}
	}
}

func (i *Info) contains(k string) bool {
	if k == "" {
		return true
	}
	_, found := i.Seen[strings.ToLower(k)]
	return found
}

func (i *Info) containsAll(m map[string]struct{}) bool {
	for k := range m {
		if !i.contains(k) {
			return false
		}
	}
	return true
}

func (i *Info) Includes(chain cssselector.Chain) bool {
	for _, s := range chain {
		contains := i.contains(s.ID) && i.contains(s.Tag) && i.containsAll(s.Class) && i.containsAll(s.Attr)
		if !contains {
			return false
		}
	}
	return true
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-'
}

func scanWords(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Skip leading non word runes.
	start := 0
	for width := 0; start < len(data); start += width {
		var r rune
		r, width = utf8.DecodeRune(data[start:])
		if isWordRune(r) {
			break
		}
	}
	// Scan until we have word runes.
	for width, i := 0, start; i < len(data); i += width {
		var r rune
		r, width = utf8.DecodeRune(data[i:])
		if !isWordRune(r) {
			return i + width, data[start:i], nil
		}
	}
	// If we're at EOF, we may have something collected.
	if atEOF && len(data) > start {
		return len(data), data[start:], nil
	}
	// Or request more data.
	return start, nil, nil
}

func Extract(r io.Reader) (*Info, error) {
	i := &Info{Seen: make(map[string]struct{})}
	scanner := bufio.NewScanner(r)
	scanner.Split(scanWords)
	for scanner.Scan() {
		i.Seen[string(bytes.ToLower(scanner.Bytes()))] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.WithStack(err)
	}
	return i, nil
}
