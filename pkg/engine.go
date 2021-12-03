package flow

import (
	"context"
	"github.com/Jeffail/tunny"
	"github.com/bytepowered/runv"
	"github.com/sirupsen/logrus"
)

var (
	_ runv.Initable = new(EventEngine)
	_ runv.Shutdown = new(EventEngine)
)

type EngineOption func(*EventEngine)

type EventEngine struct {
	workers       *tunny.Pool
	sources       map[string]SourceAdapter
	filters       []EventFilter
	transformers  []Transformer
	globalFilters []EventFilter
	dispatchers   []Dispatcher
	pipelines     []*Pipeline
}

func NewEventEngine(opts ...EngineOption) *EventEngine {
	fd := &EventEngine{
		sources:       make(map[string]SourceAdapter, 2),
		dispatchers:   make([]Dispatcher, 2),
		globalFilters: make([]EventFilter, 0, 4),
	}
	fd.workers = tunny.NewFunc(10_000, func(i interface{}) interface{} {
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
	runv.Assert(0 < len(e.sources), "sources is required")
	runv.Assert(0 < len(e.dispatchers), "dispatchers is required")
	groups := make([]PipeGroupDefine, 0)
	for _, grp := range groups {
		for _, rule := range e.flat(grp) {
			source, ok := e.sources[rule.SourceTag]
			runv.Assert(ok, "source-adapter must be found, tag: "+rule.SourceTag)
			pipe := NewPipeline(e.onPipelineEmitFunc)
			for _, gf := range e.globalFilters {
				pipe.AddEventFilter(gf)
			}
			e.compile(pipe, rule)
			e.pipelines = append(e.pipelines, pipe)
			source.SetEmitter(pipe)
		}
	}
	runv.Assert(0 < len(e.pipelines), "pipelines is required")
	return nil
}

func (e *EventEngine) compile(pipe *Pipeline, rule PipeRuledDefine) *Pipeline {
	// filters
	TagPatterns(rule.Filters).match(e.filters, func(v interface{}) {
		pipe.AddEventFilter(v.(EventFilter))
	})
	// transformer
	TagPatterns(rule.Transformers).match(e.transformers, func(v interface{}) {
		pipe.AddTransformer(v.(Transformer))
	})
	// dispatcher
	TagPatterns(rule.Dispatchers).match(e.dispatchers, func(v interface{}) {
		pipe.AddDispatcher(v.(Dispatcher))
	})
	return pipe
}

func (e *EventEngine) flat(group PipeGroupDefine) []PipeRuledDefine {
	// filters
	//for tag, f := range e.globalFilters {
	//	if match()
	//}
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

func (e *EventEngine) onPipelineEmitFunc(pipe *Pipeline, ctx EventContext, record EventRecord) {
	if ctx.Async() {
		_ = e.onPipelineWorkFunc(pipe, ctx, record)
	} else {
		e.workers.Process([]interface{}{pipe, ctx, record})
	}
}

func (e *EventEngine) Shutdown(ctx context.Context) error {
	e.xlog().Infof("shutdown")
	e.workers.Close()
	return nil
}

func (e *EventEngine) AddGlobalFilter(f EventFilter) {
	e.xlog().Infof("add global filter, tag: %s", f.Tag())
	e.globalFilters = append(e.globalFilters, f)
}

func (e *EventEngine) AddFilter(f EventFilter) {
	e.xlog().Infof("add filter, tag: %s", f.Tag())
	e.filters = append(e.filters, f)
}

func (e *EventEngine) AddDispatcher(d Dispatcher) {
	e.xlog().Infof("add dispatcher, tag: %s", d.Tag())
	e.dispatchers = append(e.dispatchers, d)
}

func (e *EventEngine) AddSourceAdapter(sa SourceAdapter) {
	e.xlog().Infof("add source-adapter, tag: %s", sa.Tag())
	e.sources[sa.Tag()] = sa
}

func (e *EventEngine) xlog() *logrus.Logger {
	return Log().WithField("app", "engine").Logger
}

func WithEventEngineWorkerSize(size uint) EngineOption {
	return func(d *EventEngine) {
		d.workers.SetSize(int(size))
	}
}
