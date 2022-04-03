package filter

import (
	flow "github.com/bytepowered/flow/v3/pkg"
	"github.com/bytepowered/runv"
	"github.com/bytepowered/runv/assert"
)

var _ flow.Filter = new(KindFilter)
var _ runv.Initable = new(KindFilter)

type KindOptions struct {
	Allows []flow.Kind `toml:"allows"`
	Drops  []flow.Kind `toml:"drops"`
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
	assert.Must(len(h.config.Opts.Allows)+len(h.config.Opts.Drops) > 0, "allows/drops is required")
	return nil
}

func (h *KindFilter) DoFilter(next flow.FilterFunc) flow.FilterFunc {
	allows := len(h.config.Opts.Allows) > 0
	drops := len(h.config.Opts.Drops) > 0
	return func(ctx flow.StateContext, event flow.Event) error {
		etype := event.Header().Kind
		if allows && h.contains(h.config.Opts.Allows, etype) {
			return next(ctx, event)
		}
		if drops && h.contains(h.config.Opts.Drops, etype) {
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
