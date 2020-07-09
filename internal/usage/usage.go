package usage

import "github.com/daaku/cssdalek/internal/cssselector"

type Info interface {
	Includes(chain cssselector.Chain) bool
}

type MultiInfo []Info

func (i MultiInfo) Includes(chain cssselector.Chain) bool {
	for _, ii := range i {
		if ii.Includes(chain) {
			return true
		}
	}
	return false
}
