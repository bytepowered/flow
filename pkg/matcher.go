package flow

import (
	"github.com/bytepowered/runv"
	"reflect"
	"strings"
)

const (
	TagGlobal = "@global"
)

type PipeGroup struct {
	Id           string   `toml:"id"`
	Sources      []string `toml:"sources"`      // 匹配Source的Tag Pattern
	Filters      []string `toml:"filters"`      // 匹配Filter的Tag Pattern
	Transformers []string `toml:"transformers"` // 匹配Transformer的Tag Pattern
	Dispatchers  []string `toml:"dispatchers"`  // 匹配Dispatcher的Tag Pattern
}

type PipeRouter struct {
	SourceTag    string
	GroupId      string
	Filters      []string
	Transformers []string
	Dispatchers  []string
}

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
