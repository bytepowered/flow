package flow

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
)

var _ Emitter = new(Router)

type GroupDescriptor struct {
	Description  string   `toml:"description"`  // 路由分组描述
	Sources      []string `toml:"sources"`      // 匹配Source的Tag Pattern
	Filters      []string `toml:"filters"`      // 匹配Filter的Tag Pattern
	Transformers []string `toml:"transformers"` // 匹配Transformer的Tag Pattern
	Outputs      []string `toml:"outputs"`      // 匹配Dispatcher的Tag Pattern
}

type router struct {
	description  string
	source       string
	filters      []string
	transformers []string
	outputs      []string
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

func NewRouterOf(emitf RouterEmitFunc) *Router {
	return &Router{
		emitter:      emitf,
		filters:      make([]Filter, 0, 2),
		transformers: make([]Transformer, 0, 2),
		outputs:      make([]Output, 0, 2),
	}
}

func (p *Router) AddFilter(f Filter) {
	p.filters = append(p.filters, f)
}

func (p *Router) AddTransformer(t Transformer) {
	p.transformers = append(p.transformers, t)
}

func (p *Router) AddOutput(d Output) {
	p.outputs = append(p.outputs, d)
}

func (p *Router) Emit(context StateContext, record Event) {
	p.emitter(p, context, record)
}

func (p *Router) route(context StateContext, inevt Event) {
	// filter -> transformer -> dispatcher
	cm := metrics()
	defer func(t *prometheus.Timer) {
		t.ObserveDuration()
	}(cm.NewTimer(inevt.Tag(), "emit"))
	cm.NewCounter(inevt.Tag(), "received").Inc()
	next := FilterFunc(func(ctx StateContext, event Event) (err error) {
		for _, tf := range p.transformers {
			event, err = tf.DoTransform(event)
			if err != nil {
				return fmt.Errorf("at transoform(:%s) error: %w", tf.Tag(), err)
			}
		}
		for _, op := range p.outputs {
			if err = op.Send(event); err != nil {
				return fmt.Errorf("on output(%s) error: %w", op.Tag(), err)
			}
		}
		return nil
	})
	if err := p.filterChainOf(next, p.filters)(context, inevt); err != nil {
		cm.NewCounter(inevt.Tag(), "error").Inc()
		Log().Errorf("route(%s->) event, error: %s", inevt.Tag(), err)
	}
}

func (p *Router) filterChainOf(next FilterFunc, filters []Filter) FilterFunc {
	for i := len(filters) - 1; i >= 0; i-- {
		next = filters[i].DoFilter(next)
	}
	return next
}
