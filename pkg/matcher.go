package flow

import (
	"github.com/bytepowered/runv/assert"
	"reflect"
	"strings"
)

type matcher []string

func newMatcher(patterns []string) matcher {
	return patterns
}

func (m matcher) on(plugins interface{}, acceptor func(interface{})) {
	vs := reflect.ValueOf(plugins)
	assert.Must(vs.Kind() == reflect.Slice, "'plugins' must be a slice")
	for i := 0; i < vs.Len(); i++ {
		elev := vs.Index(i)
		objv := elev.Interface()
		plg, ok := objv.(Plugin)
		assert.Must(ok, "'plugins' values must be typeof 'Plugin'")
		tag := plg.Tag()
		for _, pattern := range m {
			if match0(pattern, tag) {
				acceptor(objv)
			}
		}
	}
}

func match0(pattern, tag string) bool {
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
