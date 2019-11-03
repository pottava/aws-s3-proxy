package controllers

import (
	"strings"
	"unicode"
)

type s3objects []string

func (s s3objects) Len() int {
	return len(s)
}
func (s s3objects) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s s3objects) Less(i, j int) bool {
	if strings.Contains(s[i], "/") {
		if !strings.Contains(s[j], "/") {
			return true
		}
	} else {
		if strings.Contains(s[j], "/") {
			return false
		}
	}
	irs := []rune(s[i])
	jrs := []rune(s[j])

	max := len(irs)
	if max > len(jrs) {
		max = len(jrs)
	}
	for idx := 0; idx < max; idx++ {
		ir := irs[idx]
		jr := jrs[idx]
		irl := unicode.ToLower(ir)
		jrl := unicode.ToLower(jr)

		if irl != jrl {
			return irl < jrl
		}
		if ir != jr {
			return ir < jr
		}
	}
	return false
}
