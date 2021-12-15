package flow

import (
	"context"
	"fmt"
	"github.com/Jeffail/tunny"
	"github.com/bytepowered/runv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	_ runv.Initable = new(EventEngine)
	_ runv.Liveness = new(EventEngine)
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
	if err := viper.UnmarshalKey("router", &groups); err != nil {
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

func (e *EventEngine) flat(group GroupDescriptor) []router {
	routers := make([]router, 0, len(e._sources))
	TagMatcher(group.Sources).match(e._sources, func(v interface{}) {
		routers = append(routers, router{
			source:       v.(Source).Tag(),
			description:  group.Description,
			filters:      group.Filters,
			transformers: group.Transformers,
			outputs:      group.Outputs,
		})
	})
	return routers
}

func (e *EventEngine) compile(groups []GroupDescriptor) {
	for _, group := range groups {
		for _, rw := range e.flat(group) {
			nr := NewRouterOf(e.doAsyncEmitFunc)
			Log().Infof("bind router, src.tag: %s, route: %+v, group: %s", rw.source, rw, group.Description)
			e.Bind(rw.source, e.lookup(nr, rw))
		}
	}
}

func (e *EventEngine) lookup(router *Router, rw router) *Router {
	// filters
	TagMatcher(rw.filters).match(e._filters, func(v interface{}) {
		router.AddFilter(v.(Filter))
	})
	// transformer
	TagMatcher(rw.transformers).match(e._transformers, func(v interface{}) {
		router.AddTransformer(v.(Transformer))
	})
	// output
	TagMatcher(rw.outputs).match(e._outputs, func(v interface{}) {
		router.AddOutput(v.(Output))
	})
	return router
}

func (e *EventEngine) xlog() *logrus.Logger {
	return Log().WithField("app", "engine").Logger
}

func WithWorkerSize(size uint) EventEngineOption {
	return func(d *EventEngine) {
		d.coroutines.SetSize(int(size))
	}
}
