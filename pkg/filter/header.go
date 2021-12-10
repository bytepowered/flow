package filter

import (
	flow "github.com/bytepowered/flow/v2/pkg"
	"github.com/bytepowered/runv"
)

var _ flow.Filter = new(HeaderFilter)
var _ runv.Initable = new(HeaderFilter)
var _ runv.Disabled = new(HeaderFilter)

type HeaderOptions struct {
	Allowed  []int `toml:"allow-event-types"`
	Rejected []int `toml:"reject-event-types"`
}

type HeaderConfig struct {
	flow.BasedConfiguration `toml:",squash"`
	Opts                    HeaderOptions `toml:"configuration"`
}

type HeaderFilter struct {
	config HeaderConfig
}

func (h *HeaderFilter) TypeId() string {
	return "@skip.header"
}

func (h *HeaderFilter) Tag() string {
	return flow.TagGLOBAL + "filter"
}

func (h *HeaderFilter) Disabled() (reason string, disable bool) {
	return flow.LookupStateOf(flow.ConfigRootFilter, h.TypeId())
}

func (h *HeaderFilter) OnInit() error {
	_, err := flow.RootConfigOf(flow.ConfigRootFilter).Lookup(h.TypeId(), &h.config)
	if err != nil {
		return err
	}
	runv.Assert(len(h.config.Opts.Allowed)+len(h.config.Opts.Rejected) > 0, "allows/rejects is required")
	return nil
}

func (h *HeaderFilter) DoFilter(next flow.FilterFunc) flow.FilterFunc {
	allowed := len(h.config.Opts.Allowed) > 0
	rejected := len(h.config.Opts.Rejected) > 0
	return func(ctx flow.StateContext, record flow.Event) error {
		etype := int(record.Header().Kind)
		if allowed && h.contains(h.config.Opts.Allowed, etype) {
			return next(ctx, record)
		}
		if rejected && h.contains(h.config.Opts.Rejected, etype) {
			return nil
		}
		return next(ctx, record)
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
