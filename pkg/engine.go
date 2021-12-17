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
	_sources      []Source
	_filters      []Filter
	_transformers []Transformer
	_outputs      []Output
	_routers      []*Router
}

func NewEventEngine(opts ...EventEngineOption) *EventEngine {
	engine := &EventEngine{}
	engine.coroutines = tunny.NewFunc(1000, func(i interface{}) interface{} {
		args := i.([]interface{})
		nr, ctx, event := args[0].(*Router), args[1].(StateContext), args[2].(Event)
		return engine.onRouterWorkFunc(nr, ctx, event)
	})
	for _, opt := range opts {
		opt(engine)
	}
	runv.AddPostHook(engine.statechk)
	return engine
}

func (e *EventEngine) OnInit() error {
	e.xlog().Infof("init")
	runv.Assert(0 < len(e._sources), "engine.sources is required")
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

func (e *EventEngine) Bind(sourceTag string, router *Router) {
	// find source
	source, ok := func(tag string) (Source, bool) {
		for _, s := range e._sources {
			if tag == s.Tag() {
				return s, true
			}
		}
		return nil, false
	}(sourceTag)
	runv.Assert(ok, "source is required, source.tag: "+sourceTag)
	runv.Assert(len(router.outputs) > 0, "outputs is required, source.tag: "+sourceTag)
	// bind async emit func
	if router.emitter == nil {
		router.emitter = e.doAsyncEmitFunc
	}
	// bind source
	source.AddEmitter(router)
	e._routers = append(e._routers, router)
	e.xlog().Infof("bind router, source.tag: %s", sourceTag)
}

func (e *EventEngine) onRouterWorkFunc(router *Router, ctx StateContext, record Event) error {
	defer func() {
		if err := recover(); err != nil {
			e.xlog().Errorf("on route event, unexcepted panic: %s, event.tag: %s, event.type: %s", err, record.Tag(), record.Header().Kind.String())
		}
	}()
	router.route(ctx, record)
	return nil
}

func (e *EventEngine) doAsyncEmitFunc(router *Router, ctx StateContext, event Event) {
	if ctx.State().Is(StateAsync) {
		_ = e.onRouterWorkFunc(router, ctx, event)
	} else {
		e.coroutines.Process([]interface{}{router, ctx, event})
	}
}

func (e *EventEngine) statechk() error {
	runv.Assert(0 < len(e._routers), "engine routers is required")
	return nil
}

func (e *EventEngine) SetSources(v []Source) {
	e._sources = v
}

func (e *EventEngine) SetOutput(v []Output) {
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
	routers := make([]router, 0, len(e._sources))
	TagMatcher(group.Selector.SourceTags).match(e._sources, func(v interface{}) {
		routers = append(routers, router{
			description:     group.Description,
			SourceTag:       v.(Source).Tag(),
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
		runv.Assert(len(group.Selector.SourceTags) > 0, "router group, selector 'source-tags' is required")
		runv.Assert(len(group.Selector.OutputTags) > 0, "router group, selector 'output-tags' is required")
		verify := func(tags []string, msg, src string) {
			for _, t := range tags {
				runv.Assert(len(t) >= 3, msg, t, src)
			}
		}
		for _, rw := range e.flat(group) {
			runv.Assert(len(rw.SourceTag) >= 3, "router, 'source-tag' is invalid, tag: "+rw.SourceTag+", group: "+group.Description)
			verify(rw.FilterTags, "router, 'filter-tag' is invalid, tag: %s, src: %s", rw.SourceTag)
			verify(rw.TransformerTags, "router, 'transformer-tag' is invalid, tag: %s, src: %s", rw.SourceTag)
			verify(rw.OutputTags, "router, 'output-tag' is invalid, tag: %s, src: %s", rw.SourceTag)
			nr := NewRouterOf(e.doAsyncEmitFunc)
			Log().Infof("bind router, src.tag: %s, route: %+v, desc: %s", rw.SourceTag, rw, group.Description)
			e.Bind(rw.SourceTag, e.lookup(nr, rw))
		}
	}
}

func (e *EventEngine) lookup(router *Router, rw router) *Router {
	// filters
	TagMatcher(rw.FilterTags).match(e._filters, func(v interface{}) {
		router.AddFilter(v.(Filter))
	})
	// transformer
	TagMatcher(rw.TransformerTags).match(e._transformers, func(v interface{}) {
		router.AddTransformer(v.(Transformer))
	})
	// output
	TagMatcher(rw.OutputTags).match(e._outputs, func(v interface{}) {
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
