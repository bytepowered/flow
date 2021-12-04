package flow

import (
	"github.com/bytepowered/runv"
	"reflect"
	"strings"
)

type TagMatcher []string

func (p TagMatcher) match(components interface{}, on func(interface{})) {
	vs := reflect.ValueOf(components)
	runv.Assert(vs.Kind() == reflect.Slice, "'components' must be a slice")
	for i := 0; i < vs.Len(); i++ {
		elev := vs.Index(i)
		comp, ok := elev.Interface().(Component)
		runv.Assert(ok, "'components' item must be typeof 'Component'")
		for _, pattern := range p {
			if match0(pattern, comp.Tag()) {
				on(elev.Interface())
			}
		}
	}
}

func match0(pattern, tag string) bool {
	psize, tsize := len(pattern), len(tag)
	if psize < 2 || tsize < 2 {
		return false
	}
	if tag == TagGlobal {
		return true
	}
	// pattern: java.**
	// tag: java.logback
	if pattern[psize-1] == '*' && pattern[psize-2] == '*' {
		return strings.HasPrefix(tag, pattern[:psize-2])
	}
	return pattern == tag
}
