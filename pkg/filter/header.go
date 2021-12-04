package filter

import (
	flow "github.com/bytepowered/flow/pkg"
	"github.com/bytepowered/runv"
)

var _ flow.EventFilter = new(HeaderFilter)
var _ runv.Initable = new(HeaderFilter)

type HeaderOptions struct {
	Allowed  []int `toml:"allow-event-types"`
	Rejected []int `toml:"reject-event-types"`
}

type HeaderFilter struct {
	priority *int
	opts     HeaderOptions
}

func (h *HeaderFilter) Priority() int {
	if h.priority == nil {
		return -10099
	}
	return *h.priority
}

func (h *HeaderFilter) TypeId() string {
	return "@skip.header"
}

func (h *HeaderFilter) Tag() string {
	return flow.TagGlobal
}

func (h *HeaderFilter) OnInit() error {
	//
	return nil
}

func (h *HeaderFilter) DoFilter(next flow.EventFilterFunc) flow.EventFilterFunc {
	allowed := len(h.opts.Allowed) > 0
	rejected := len(h.opts.Rejected) > 0
	return func(ctx flow.EventContext, record flow.EventRecord) error {
		etype := int(record.Header().Type)
		switch {
		case allowed && h.contains(h.opts.Allowed, etype):
			return next(ctx, record)
		case rejected && h.contains(h.opts.Rejected, etype):
			return nil // discard
		default:
			return next(ctx, record)
		}
	}
}

func (h *HeaderFilter) contains(values []int, in int) bool {
	for _, v := range values {
		if v == in {
			return true
		}
	}
	return false
}
