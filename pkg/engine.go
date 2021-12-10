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
}

func NewEventEngine(opts ...EventEngineOption) *EventEngine {
	fd := &EventEngine{}
	fd.coroutines = tunny.NewFunc(1000, func(i interface{}) interface{} {
		args := i.([]interface{})
		router, ctx, event := args[0].(*Router), args[1].(StateContext), args[2].(Event)
		return fd.onRouterWorkFunc(router, ctx, event)
	})
	for _, opt := range opts {
		opt(fd)
	}
	return fd
}

func (e *EventEngine) OnInit() error {
	e.xlog().Infof("init")
	runv.Assert(0 < len(e._sources), "engine.sources is required")
	runv.Assert(0 < len(e._outputs), "engine.outputs is required")
	groups := make([]GroupRouterW, 0)
	if err := viper.UnmarshalKey("routers", &groups); err != nil {
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
	runv.Assert(ok, "router.sources is required, source.tag: "+sourceTag)
	runv.Assert(len(router.outputs) > 0, "router.outputs is required, source.tag: "+sourceTag)
	// bind work func
	if router.emitter == nil {
		router.emitter = e.doAsyncEmitFunc
	}
	// bind source adapter
	source.SetEmitter(router)
	e.xlog().Infof("bind router, source.tag: %s", sourceTag)
}

func (e *EventEngine) onRouterWorkFunc(pipe *Router, ctx StateContext, record Event) error {
	defer func() {
		if err := recover(); err != nil {
			e.xlog().Errorf("match routers work, unexcepted panic error: %s, event.tag: %s, event.type: %s", err, record.Tag(), record.Header().Kind.String())
		}
	}()
	pipe.doEmit(ctx, record)
	return nil
}

func (e *EventEngine) doAsyncEmitFunc(router *Router, ctx StateContext, record Event) {
	if ctx.State().Is(StateAsync) {
		_ = e.onRouterWorkFunc(router, ctx, record)
	} else {
		e.coroutines.Process([]interface{}{router, ctx, record})
	}
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

func (e *EventEngine) flat(group GroupRouterW) []RouterW {
	routers := make([]RouterW, 0, len(e._sources))
	TagMatcher(group.Sources).match(e._sources, func(v interface{}) {
		routers = append(routers, RouterW{
			SourceTag:    v.(Source).Tag(),
			GroupId:      group.GroupId,
			Filters:      group.Filters,
			Transformers: group.Transformers,
			Outputs:      group.Outputs,
		})
	})
	return routers
}

func (e *EventEngine) compile(groups []GroupRouterW) {
	for _, grp := range groups {
		for _, router := range e.flat(grp) {
			pipe := NewRouterOf(e.doAsyncEmitFunc)
			e.Bind(router.SourceTag, e.lookup(pipe, router))
		}
	}
}

func (e *EventEngine) lookup(router *Router, rw RouterW) *Router {
	// filters
	TagMatcher(rw.Filters).match(e._filters, func(v interface{}) {
		router.AddFilter(v.(Filter))
	})
	// transformer
	TagMatcher(rw.Transformers).match(e._transformers, func(v interface{}) {
		router.AddTransformer(v.(Transformer))
	})
	// output
	TagMatcher(rw.Outputs).match(e._outputs, func(v interface{}) {
		router.AddOutput(v.(Output))
	})
	return router
}

func (e *EventEngine) xlog() *logrus.Logger {
	return Log().WithField("app", "engine").Logger
}

func WithEventEngineWorkerSize(size uint) EventEngineOption {
	return func(d *EventEngine) {
		d.coroutines.SetSize(int(size))
	}
}
