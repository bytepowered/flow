package filter

import (
	flow "github.com/bytepowered/flow/v3/pkg"
	"github.com/bytepowered/runv"
	"github.com/bytepowered/runv/assert"
)

var _ flow.Filter = new(KindFilter)
var _ runv.Initable = new(KindFilter)

type KindOptions struct {
	Allowed  []flow.Kind `toml:"allow-kinds"`
	Rejected []flow.Kind `toml:"reject-kinds"`
}

type KindConfig struct {
	flow.BasedConfiguration `toml:",squash"`
	Opts                    KindOptions `toml:"configuration"`
}

type KindFilter struct {
	config KindConfig
}

func NewKindFilter() *KindFilter {
	return &KindFilter{}
}

func (h *KindFilter) Tag() string {
	return h.config.Tag
}

func (h *KindFilter) OnInit() error {
	assert.Must(len(h.config.Opts.Allowed)+len(h.config.Opts.Rejected) > 0, "allows/rejects is required")
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

func (h *KindFilter) Update(c KindConfig) *KindFilter {
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
