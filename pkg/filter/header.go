package filter

import (
	"fmt"
	flow "github.com/bytepowered/flow/v2/pkg"
	"github.com/bytepowered/runv"
)

var _ flow.EventFilter = new(HeaderFilter)
var _ runv.Initable = new(HeaderFilter)
var _ runv.Activable = new(HeaderFilter)

type HeaderConfig struct {
	flow.BasicConfig `toml:",squash"`
	Options          HeaderOptions `toml:"configuration"`
}

type HeaderOptions struct {
	Allowed  []int `toml:"allow-event-types"`
	Rejected []int `toml:"reject-event-types"`
}

type HeaderFilter struct {
	priority *int
	opts     HeaderOptions
}

func (h *HeaderFilter) TypeId() string {
	return "@skip.header"
}

func (h *HeaderFilter) Tag() string {
	return flow.TagGlobal
}

func (h *HeaderFilter) Active() bool {
	return flow.ConfigurationsOf(flow.ConfigKeyFilter).ActiveOf(h.TypeId())
}

func (h *HeaderFilter) OnInit() error {
	config := HeaderConfig{}
	err := flow.ConfigurationsOf(flow.ConfigKeyFilter).MarshalLookup(h.TypeId(), &config)
	if err != nil {
		return fmt.Errorf("load config error, type: %s, error: %w", h.TypeId(), err)
	}
	h.opts = config.Options
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
