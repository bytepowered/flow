package flow

import (
	"github.com/bytepowered/runv"
	"reflect"
	"strings"
)

type TagMatcher []string

func (tm TagMatcher) match(components interface{}, on func(interface{})) {
	vs := reflect.ValueOf(components)
	runv.Assert(vs.Kind() == reflect.Slice, "'components' must be a slice")
	for i := 0; i < vs.Len(); i++ {
		elev := vs.Index(i)
		objv := elev.Interface()
		plg, ok := objv.(Plugin)
		runv.Assert(ok, "'components' values must be typeof 'Plugin'")
		tag := plg.Tag()
		for _, pattern := range tm {
			if match0(pattern, tag) {
				on(objv)
			}
		}
	}
}

func match0(pattern, tag string) bool {
	psize, tsize := len(pattern), len(tag)
	if psize < 2 || tsize < 2 {
		return false
	}
	// pattern: java.* --> tag: java.logback
	if pattern[psize-1] == '*' {
		return strings.HasPrefix(tag, pattern[:psize-1])
	}
	return pattern == tag
}
