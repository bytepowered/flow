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
	// 注意：基于Components来迭代，匹配声明为Global的组件；
	for i := 0; i < vs.Len(); i++ {
		elev := vs.Index(i)
		objv := elev.Interface()
		comp, ok := objv.(Component)
		runv.Assert(ok, "'components' item must be typeof 'Component'")
		tag := comp.Tag()
		// globals
		if global0(tag) {
			on(objv)
			continue
		}
		// if match
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
	// pattern: java.**
	// tag: java.logback
	if pattern[psize-1] == '*' && pattern[psize-2] == '*' {
		return strings.HasPrefix(tag, pattern[:psize-2])
	}
	return pattern == tag
}

func global0(tag string) bool {
	return strings.HasPrefix(tag, TagGLOBAL)
}
