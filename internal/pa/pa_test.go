package pa

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/daaku/ensure"
	"github.com/pkg/errors"
)

func TestFinishSuccess(t *testing.T) {
	err := Finish(func() Next { return nil })
	ensure.Nil(t, err)
}

func TestFinishError(t *testing.T) {
	err := Finish(func() Next { panic(os.ErrClosed) })
	ensure.True(t, errors.Is(err, os.ErrClosed))
}

func TestFinishPanic(t *testing.T) {
	const v = 42
	defer ensure.PanicDeepEqual(t, v)
	Finish(func() Next { panic(v) })
}

func TestWriteStringError(t *testing.T) {
	f, err := ioutil.TempFile("", "cssdalek-pa-")
	ensure.Nil(t, err)
	f.Close()
	os.Remove(f.Name())

	defer func() {
		r := recover()
		ensure.True(t, errors.Is(r.(error), os.ErrClosed))
	}()
	WriteString(f, "a")
}

func TestWriteError(t *testing.T) {
	f, err := ioutil.TempFile("", "cssdalek-pa-")
	ensure.Nil(t, err)
	f.Close()
	os.Remove(f.Name())

	defer func() {
		r := recover()
		ensure.True(t, errors.Is(r.(error), os.ErrClosed))
	}()
	Write(f, []byte("a"))
}
