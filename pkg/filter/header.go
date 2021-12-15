package filter

import (
	flow "github.com/bytepowered/flow/v2/pkg"
	"github.com/bytepowered/runv"
)

var _ flow.Filter = new(KindFilter)
var _ runv.Initable = new(KindFilter)
var _ runv.Disabled = new(KindFilter)

type KindOptions struct {
	Allowed  []flow.Kind `toml:"allow-kinds"`
	Rejected []flow.Kind `toml:"reject-kinds"`
}

type KindConfig struct {
	flow.BasedConfiguration `toml:",squash"`
	Opts                    KindOptions `toml:"configuration"`
}

type KindFilter struct {
	*GlobalFilter
	config KindConfig
}

func NewKindFilter() *KindFilter {
	return &KindFilter{
		GlobalFilter: &GlobalFilter{
			TypeTag: "skip.kind",
		},
	}
}

func (h *KindFilter) Disabled() (reason string, disable bool) {
	return flow.LookupStateOf(flow.ConfigRootFilter, h.TypeId())
}

func (h *KindFilter) OnInit() error {
	_, err := flow.RootConfigOf(flow.ConfigRootFilter).Lookup(h.TypeId(), &h.config)
	if err != nil {
		return err
	}
	runv.Assert(len(h.config.Opts.Allowed)+len(h.config.Opts.Rejected) > 0, "allows/rejects is required")
	return nil
}

func (h *KindFilter) DoFilter(next flow.FilterFunc) flow.FilterFunc {
	allowed := len(h.config.Opts.Allowed) > 0
	rejected := len(h.config.Opts.Rejected) > 0
	return func(ctx flow.StateContext, event flow.Event) error {
		etype := event.Header().Kind
		if allowed && h.contains(h.config.Opts.Allowed, etype) {
			return next(ctx, event)
		}
		if rejected && h.contains(h.config.Opts.Rejected, etype) {
			return nil
		}
		return next(ctx, event)
	}
}

func (h *KindFilter) UpdateConfig(c KindConfig) *KindFilter {
	h.config = c
	return h
}

func (h *KindFilter) contains(values []flow.Kind, in flow.Kind) bool {
	for _, v := range values {
		if v == in {
			return true
		}
	}
	return false
}
