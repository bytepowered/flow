package flow

import (
	"github.com/bytepowered/runv"
	"reflect"
	"strings"
)

type PipeGroupDefine struct {
	Id           string   `toml:"id"`
	Sources      []string `toml:"sources"`
	Filters      []string `toml:"filters"`
	Transformers []string `toml:"transformers"`
	Dispatchers  []string `toml:"dispatchers"`
}

type PipeRuledDefine struct {
	SourceTag    string
	RuleId       string   `toml:"id"`
	Filters      []string `toml:"filters"`
	Transformers []string `toml:"transformers"`
	Dispatchers  []string `toml:"dispatchers"`
}

type TagPatterns []string

func (p TagPatterns) match(components interface{}, on func(interface{})) {
	vs := reflect.ValueOf(components)
	runv.Assert(vs.Kind() == reflect.Slice, "'components' must be a slice")
	for i := 0; i < vs.Len(); i++ {
		elev := vs.Index(i)
		comp, ok := elev.Interface().(Component)
		runv.Assert(ok, "'components' item must be typeof 'Component'")
		for _, pattern := range p {
			if match(pattern, comp.Tag()) {
				on(elev.Interface())
			}
		}
	}
}

func match(pattern, tag string) bool {
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
