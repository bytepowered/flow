package flow

import (
	"strings"
)

func lookup[T Pluginable](patterns []string, plugins []T, onfound func(tag string, plg T)) {
	matches(patterns, plugins, onfound, nil)
}

func matches[T Pluginable](patterns []string, plugins []T, onfound func(tag string, plg T), onmissed func(tag string)) {
	for _, plg := range plugins {
		for _, pattern := range patterns {
			if match(pattern, plg.Tag()) {
				onfound(plg.Tag(), plg)
			} else if onmissed != nil {
				onmissed(plg.Tag())
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
