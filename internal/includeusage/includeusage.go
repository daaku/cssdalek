package includeusage

import (
	"regexp"

	"github.com/daaku/cssdalek/internal/cssselector"
)

type IncludeClass struct {
	Re []*regexp.Regexp
}

func (i *IncludeClass) Includes(chain cssselector.Chain) bool {
outer:
	for _, s := range chain {
		for _, r := range i.Re {
			for c := range s.Class {
				if r.MatchString(c) {
					continue outer
				}
			}
		}
		return false
	}
	return true
}

type IncludeID struct {
	Re []*regexp.Regexp
}

func (i *IncludeID) Includes(chain cssselector.Chain) bool {
outer:
	for _, s := range chain {
		for _, r := range i.Re {
			if r.MatchString(s.ID) {
				continue outer
			}
		}
		return false
	}
	return true
}
