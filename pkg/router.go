package flow

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
)

var _ Emitter = new(Router)

type GroupDescriptor struct {
	Description string `toml:"description"` // 路由分组描述
	Selector    struct {
		InputTags       []string `toml:"inputs"`       // 匹配Input的Tag Pattern
		FilterTags      []string `toml:"filters"`      // 匹配Filter的Tag Pattern
		TransformerTags []string `toml:"transformers"` // 匹配Transformer的Tag Pattern
		OutputTags      []string `toml:"outputs"`      // 匹配Dispatcher的Tag Pattern
	} `toml:"selector"`
}

type router struct {
	description     string
	InputTag        string
	FilterTags      []string
	TransformerTags []string
	OutputTags      []string
}

type RouterEmitFunc func(*Router, StateContext, Event)

type Router struct {
	emitter      RouterEmitFunc
	filters      []Filter
	transformers []Transformer
	outputs      []Output
}

func NewRouter() *Router {
	return NewRouterOf(nil)
}

func NewRouterOf(emit RouterEmitFunc) *Router {
	return &Router{
		emitter:      emit,
		filters:      make([]Filter, 0, 2),
		transformers: make([]Transformer, 0, 2),
		outputs:      make([]Output, 0, 2),
	}
}

func (r *Router) AddFilter(f Filter) {
	r.filters = append(r.filters, f)
}

func (r *Router) AddTransformer(t Transformer) {
	r.transformers = append(r.transformers, t)
}

func (r *Router) AddOutput(d Output) {
	r.outputs = append(r.outputs, d)
}

func (r *Router) Emit(context StateContext, event Event) {
	r.emitter(r, context, event)
}

func (r *Router) route(context StateContext, inevt Event) {
	// filter -> transformer -> output
	cm := metrics()
	defer func(t *prometheus.Timer) {
		t.ObserveDuration()
	}(cm.NewTimer(inevt.Tag(), "emit"))
	cm.NewCounter(inevt.Tag(), "received").Inc()
	next := FilterFunc(func(ctx StateContext, event Event) (err error) {
		for _, trans := range r.transformers {
			event, err = trans.DoTransform(event)
			if err != nil {
				return fmt.Errorf("at transoform(:%s) error: %w", trans.Tag(), err)
			}
		}
		for _, output := range r.outputs {
			if err = output.Send(event); err != nil {
				return fmt.Errorf("on output(%s) error: %w", output.Tag(), err)
			}
		}
		return nil
	})
	if err := r.filterChainOf(next, r.filters)(context, inevt); err != nil {
		cm.NewCounter(inevt.Tag(), "error").Inc()
		Log().Errorf("route(%s->) event, error: %s", inevt.Tag(), err)
	}
}

func (r *Router) filterChainOf(next FilterFunc, filters []Filter) FilterFunc {
	for i := len(filters) - 1; i >= 0; i-- {
		next = filters[i].DoFilter(next)
	}
	return next
}
