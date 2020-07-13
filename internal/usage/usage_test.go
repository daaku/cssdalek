package usage

import (
	"testing"

	"github.com/daaku/cssdalek/internal/cssselector"
	"github.com/daaku/ensure"
)

type includer bool

func (i includer) Includes(chain cssselector.Chain) bool {
	return bool(i)
}

func TestMultiNotInclude(t *testing.T) {
	ensure.False(t, MultiInfo{includer(false)}.Includes(cssselector.Chain{}))
}

func TestMultiInclude(t *testing.T) {
	ensure.True(t, MultiInfo{includer(true)}.Includes(cssselector.Chain{}))
}

func TestMultiIncludeSecond(t *testing.T) {
	mi := MultiInfo{
		includer(false),
		includer(true),
	}
	ensure.True(t, mi.Includes(cssselector.Chain{}))
}
