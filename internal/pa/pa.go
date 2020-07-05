// Package pa is pa.
package pa

import (
	"io"

	"github.com/pkg/errors"
)

func WriteString(w io.Writer, s string) {
	if _, err := io.WriteString(w, s); err != nil {
		panic(errors.WithStack(err))
	}
}

func Write(w io.Writer, b []byte) {
	if _, err := w.Write(b); err != nil {
		panic(errors.WithStack(err))
	}
}

type Next func() Next

func Finish(n Next) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if pe, ok := r.(error); ok {
				err = pe
				return
			}
			panic(r)
		}
	}()
	for {
		if n == nil {
			break
		}
		n = n()
	}
	return
}
