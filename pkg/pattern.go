package flow

import (
	"strings"
)

func matches[T Pluginable](patterns []string, plugins []T, consumer func(tag string, plg T)) {
	for _, plg := range plugins {
		for _, pattern := range patterns {
			if match(pattern, plg.Tag()) {
				consumer(plg.Tag(), plg)
			}
		}
	}
}

func match(pattern, tag string) bool {
	psize, tsize := len(pattern), len(tag)
	if psize == 1 && '*' == pattern[0] {
		return true
	}
	if psize < 2 || tsize < 2 {
		return false
	}
	// pattern: java.* --> tag: java.logback
	if pattern[psize-1] == '*' {
		return strings.HasPrefix(tag, pattern[:psize-1])
	}
	return pattern == tag
}
