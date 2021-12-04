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
	_ runv.Shutdown = new(EventEngine)
)

type EngineOption func(*EventEngine)

type EventEngine struct {
	coroutines    *tunny.Pool
	sourcems      map[string]SourceAdapter
	_sources      []SourceAdapter
	_filters      []EventFilter
	_transformers []Transformer
	_dispatchers  []Dispatcher
}

func NewEventEngine(opts ...EngineOption) *EventEngine {
	fd := &EventEngine{
		sourcems: make(map[string]SourceAdapter, 4),
	}
	fd.coroutines = tunny.NewFunc(10_000, func(i interface{}) interface{} {
		args := i.([]interface{})
		pipe, ctx, record := args[0].(*Pipeline), args[1].(EventContext), args[2].(EventRecord)
		return fd.onPipelineWorkFunc(pipe, ctx, record)
	})
	for _, opt := range opts {
		opt(fd)
	}
	return fd
}

func (e *EventEngine) OnInit() error {
	e.xlog().Infof("init")
	runv.Assert(0 < len(e._sources), "sources is required")
	runv.Assert(0 < len(e._dispatchers), "dispatchers is required")
	groups := make([]GroupedPipelineW, 0)
	if err := viper.UnmarshalKey("pipeline", &groups); err != nil {
		return fmt.Errorf("load 'pipeline' config error: %w", err)
	}
	e.compile(groups)
	e.xlog().Infof("init load pipeline groups: %d", len(groups))
	return nil
}

func (e *EventEngine) onPipelineWorkFunc(pipe *Pipeline, ctx EventContext, record EventRecord) error {
	defer func() {
		if err := recover(); err != nil {
			e.xlog().Errorf("match pipeline work, unexcepted panic error: %s, event.tag: %s, event.type: %s", err, record.Tag(), record.Header().Type.String())
		}
	}()
	pipe.doEmit(ctx, record)
	return nil
}

func (e *EventEngine) doAsyncPipelineEmitFunc(pipe *Pipeline, ctx EventContext, record EventRecord) {
	if ctx.State().Is(EventStateAsync) {
		_ = e.onPipelineWorkFunc(pipe, ctx, record)
	} else {
		e.coroutines.Process([]interface{}{pipe, ctx, record})
	}
}

func (e *EventEngine) Shutdown(ctx context.Context) error {
	e.xlog().Infof("shutdown")
	e.coroutines.Close()
	return nil
}

func (e *EventEngine) BindPipeline(sourceTag string, pipe *Pipeline) {
	source, ok := e.sourcems[sourceTag]
	runv.Assert(ok, "source-adapter must be found, tag: "+sourceTag)
	// bind work func
	if pipe.emitf == nil {
		pipe.emitf = e.doAsyncPipelineEmitFunc
	}
	// bind source adapter
	source.SetEmitter(pipe)
	e.xlog().Infof("bind pipeline, source.tag: %s", sourceTag)
}

func (e *EventEngine) SetSourceAdapters(v []SourceAdapter) {
	e._sources = v
	for _, s := range v {
		e.sourcems[s.Tag()] = s
	}
}

func (e *EventEngine) SetDispatchers(v []Dispatcher) {
	e._dispatchers = v
}

func (e *EventEngine) SetEventFilters(v []EventFilter) {
	e._filters = v
}

func (e *EventEngine) SetTransformers(v []Transformer) {
	e._transformers = v
}

func (e *EventEngine) flat(group GroupedPipelineW) []RoutedPipelineW {
	routers := make([]RoutedPipelineW, 0, len(e.sourcems))
	TagMatcher(group.Sources).match(e._sources, func(v interface{}) {
		routers = append(routers, RoutedPipelineW{
			SourceTag:    v.(SourceAdapter).Tag(),
			GroupId:      group.GroupId,
			Filters:      group.Filters,
			Transformers: group.Transformers,
			Dispatchers:  group.Dispatchers,
		})
	})
	return routers
}

func (e *EventEngine) compile(groups []GroupedPipelineW) {
	for _, grp := range groups {
		for _, router := range e.flat(grp) {
			pipe := NewPipelineOf(e.doAsyncPipelineEmitFunc)
			e.BindPipeline(router.SourceTag, e.lookup(pipe, router))
		}
	}
}

func (e *EventEngine) lookup(pipe *Pipeline, router RoutedPipelineW) *Pipeline {
	// filters
	TagMatcher(router.Filters).match(e._filters, func(v interface{}) {
		pipe.AddEventFilter(v.(EventFilter))
	})
	// transformer
	TagMatcher(router.Transformers).match(e._transformers, func(v interface{}) {
		pipe.AddTransformer(v.(Transformer))
	})
	// dispatcher
	TagMatcher(router.Dispatchers).match(e._dispatchers, func(v interface{}) {
		pipe.AddDispatcher(v.(Dispatcher))
	})
	return pipe
}

func (e *EventEngine) xlog() *logrus.Logger {
	return Log().WithField("app", "engine").Logger
}

func WithEventEngineWorkerSize(size uint) EngineOption {
	return func(d *EventEngine) {
		d.coroutines.SetSize(int(size))
	}
}
