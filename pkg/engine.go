package flow

import (
	"context"
	"fmt"
	"github.com/Jeffail/tunny"
	"github.com/bytepowered/runv"
	"github.com/sirupsen/logrus"
)

var (
	_ runv.Initable  = new(EventEngine)
	_ runv.Liveness  = new(EventEngine)
	_ runv.Liveorder = new(EventEngine)
)

type EventEngineOption func(*EventEngine)

type EventEngine struct {
	coroutines    *tunny.Pool
	_inputs       []Input
	_filters      []Filter
	_transformers []Transformer
	_outputs      []Output
	_routers      []*Router
}

func NewEventEngine(opts ...EventEngineOption) *EventEngine {
	engine := &EventEngine{}
	engine.coroutines = tunny.NewFunc(1000, func(i interface{}) interface{} {
		vals := i.([]interface{})
		return engine.onEmitRouter(
			vals[0].(*Router),
			vals[1].(StateContext),
			vals[2].(Event))
	})
	for _, opt := range opts {
		opt(engine)
	}
	runv.AddPostHook(engine.statechk)
	return engine
}

func (e *EventEngine) OnInit() error {
	e.xlog().Infof("init")
	runv.Assert(0 < len(e._inputs), "engine.inputs is required")
	runv.Assert(0 < len(e._outputs), "engine.outputs is required")
	groups := make([]GroupDescriptor, 0)
	if err := UnmarshalKey("router", &groups); err != nil {
		return fmt.Errorf("load 'routers' config error: %w", err)
	}
	e.compile(groups)
	e.xlog().Infof("init load router groups: %d", len(groups))
	return nil
}

func (e *EventEngine) Startup(ctx context.Context) error {
	e.xlog().Infof("startup")
	return nil
}

func (e *EventEngine) Shutdown(ctx context.Context) error {
	e.xlog().Infof("shutdown")
	e.coroutines.Close()
	return nil
}

func (e *EventEngine) Bind(inputTag string, router *Router) {
	// find input
	input, ok := func(tag string) (Input, bool) {
		for _, s := range e._inputs {
			if tag == s.Tag() {
				return s, true
			}
		}
		return nil, false
	}(inputTag)
	runv.Assert(ok, "bind router: input is required, input.tag: "+inputTag)
	runv.Assert(len(router.outputs) > 0, "bind router: outputs is required, input.tag: "+inputTag)
	// bind async emit func
	if router.emitter == nil {
		router.emitter = e.doEmitFunc
	}
	// bind input
	input.AddEmitter(router)
	e._routers = append(e._routers, router)
	e.xlog().Infof("bind router: OK, input.tag: %s", inputTag)
}

func (e *EventEngine) doEmitFunc(router *Router, ctx StateContext, event Event) {
	if ctx.State().Is(StateAsync) {
		_ = e.onEmitRouter(router, ctx, event)
	} else {
		e.coroutines.Process([]interface{}{router, ctx, event})
	}
}

func (e *EventEngine) onEmitRouter(router *Router, ctx StateContext, event Event) error {
	defer func(tag, kind string) {
		if r := recover(); r != nil {
			e.xlog().Errorf("on route work, unexcepted panic: %s, event.tag: %s, event.type: %s", r, tag, kind)
		}
	}(event.Tag(), event.Kind().String())
	router.route(ctx, event)
	return nil
}

func (e *EventEngine) statechk() error {
	runv.Assert(0 < len(e._routers), "engine routers is required")
	return nil
}

func (e *EventEngine) SetInputs(v []Input) {
	e._inputs = v
}

func (e *EventEngine) SetOutputs(v []Output) {
	e._outputs = v
}

func (e *EventEngine) SetFilters(v []Filter) {
	e._filters = v
}

func (e *EventEngine) SetTransformers(v []Transformer) {
	e._transformers = v
}

func (e *EventEngine) Order(state runv.State) int {
	return 100000 // 所有生命周期都靠后
}

func (e *EventEngine) flat(group GroupDescriptor) []router {
	routers := make([]router, 0, len(e._inputs))
	TagMatcher(group.Selector.InputTags).match(e._inputs, func(v interface{}) {
		routers = append(routers, router{
			description:     group.Description,
			InputTag:        v.(Input).Tag(),
			FilterTags:      group.Selector.FilterTags,
			TransformerTags: group.Selector.TransformerTags,
			OutputTags:      group.Selector.OutputTags,
		})
	})
	return routers
}

func (e *EventEngine) compile(groups []GroupDescriptor) {
	for _, group := range groups {
		runv.Assert(group.Description != "", "router group, 'description' is required")
		runv.Assert(len(group.Selector.InputTags) > 0, "router group, selector 'input-tags' is required")
		runv.Assert(len(group.Selector.OutputTags) > 0, "router group, selector 'output-tags' is required")
		verify := func(tags []string, msg, src string) {
			for _, t := range tags {
				runv.Assert(len(t) >= 3, msg, t, src)
			}
		}
		for _, rw := range e.flat(group) {
			runv.Assert(len(rw.InputTag) >= 3, "router, 'input-tag' is invalid, tag: "+rw.InputTag+", group: "+group.Description)
			verify(rw.FilterTags, "router, 'filter-tag' is invalid, tag: %s, src: %s", rw.InputTag)
			verify(rw.TransformerTags, "router, 'transformer-tag' is invalid, tag: %s, src: %s", rw.InputTag)
			verify(rw.OutputTags, "router, 'output-tag' is invalid, tag: %s, src: %s", rw.InputTag)
			nr := NewRouterOf(e.doEmitFunc)
			Log().Infof("bind router, src.tag: %s, route: %+v", rw.InputTag, rw)
			e.Bind(rw.InputTag, e.lookup(nr, rw))
		}
	}
}

func (e *EventEngine) lookup(router *Router, mr router) *Router {
	// filters
	TagMatcher(mr.FilterTags).match(e._filters, func(v interface{}) {
		router.AddFilter(v.(Filter))
	})
	// transformer
	TagMatcher(mr.TransformerTags).match(e._transformers, func(v interface{}) {
		router.AddTransformer(v.(Transformer))
	})
	// output
	TagMatcher(mr.OutputTags).match(e._outputs, func(v interface{}) {
		router.AddOutput(v.(Output))
	})
	return router
}

func (e *EventEngine) xlog() *logrus.Entry {
	return Log().WithField("app", "engine")
}

func WithCoroutineSize(size uint) EventEngineOption {
	return func(d *EventEngine) {
		d.coroutines.SetSize(int(size))
	}
}
