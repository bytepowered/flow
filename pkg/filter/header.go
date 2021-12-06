package filter

import (
	flow "github.com/bytepowered/flow/v2/pkg"
	"github.com/bytepowered/runv"
)

var _ flow.EventFilter = new(HeaderFilter)
var _ runv.Initable = new(HeaderFilter)
var _ runv.Disabled = new(HeaderFilter)

type HeaderOptions struct {
	Allowed  []int `toml:"allow-event-types"`
	Rejected []int `toml:"reject-event-types"`
}

type HeaderConfig struct {
	flow.BasedConfiguration `toml:",squash"`
	Options                 HeaderOptions `toml:"configuration"`
}

type HeaderFilter struct {
	opts HeaderOptions
}

func (h *HeaderFilter) TypeId() string {
	return "@skip.header"
}

func (h *HeaderFilter) Tag() string {
	return flow.TagGlobal + ".filter"
}

func (h *HeaderFilter) Disabled() (reason string, disable bool) {
	return flow.LookupStateOf(flow.ConfigRootFilter, h.TypeId())
}

func (h *HeaderFilter) OnInit() error {
	_, _ = flow.RootConfigOf(flow.ConfigRootFilter).Lookup(h.TypeId(), &h.opts)
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

func NewHeaderFilter() *HeaderFilter {
	return &HeaderFilter{}
}
