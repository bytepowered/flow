package flow

import (
	"context"
	"fmt"
	"github.com/bytepowered/runv"
	"github.com/bytepowered/runv/assert"
	"github.com/sirupsen/logrus"
)

var (
	_ runv.Initable  = new(EventEngine)
	_ runv.Liveness  = new(EventEngine)
	_ runv.Liveorder = new(EventEngine)
	_ runv.Servable  = new(EventEngine)
)

type EventEngineOption func(*EventEngine)

type EventEngine struct {
	_inputs       []Input
	_filters      []Filter
	_transformers []Transformer
	_outputs      []Output
	_pipeline     []*Pipeline
	queueSize     uint
	stateContext  context.Context
	stateFunc     context.CancelFunc
}

func NewEventEngine(opts ...EventEngineOption) *EventEngine {
	ctx, ctxfunc := context.WithCancel(context.Background())
	engine := &EventEngine{
		stateContext: ctx, stateFunc: ctxfunc,
		queueSize: 10,
	}
	for _, opt := range opts {
		opt(engine)
	}
	runv.AddPostHook(engine.statechk)
	return engine
}

func (e *EventEngine) OnInit() error {
	e.xlog().Infof("ENGINE: INIT")
	assert.Must(0 < len(e._inputs), "engine.inputs is required")
	assert.Must(0 < len(e._outputs), "engine.outputs is required")
	// 从配置文件中加载route配置项
	groups := make([]PipelineDefinition, 0)
	if err := UnmarshalConfigKey("pipeline", &groups); err != nil {
		return fmt.Errorf("load 'routers' config error: %w", err)
	}
	e.compile(groups)
	e.xlog().Infof("init load router groups: %d", len(groups))
	return nil
}

func (e *EventEngine) Startup(ctx context.Context) error {
	e.xlog().Infof("ENGINE: STARTUP")
	return nil
}

func (e *EventEngine) Shutdown(ctx context.Context) error {
	e.stateFunc()
	e.xlog().Infof("ENGINE: SHUTDOWN")
	return nil
}

func (e *EventEngine) Serve(c context.Context) error {
	e.xlog().Infof("ENGINE: SERVE")
	doqueue := func(ctx context.Context, tag string, queue <-chan Event) {
		e.xlog().Infof("deliver(%s): queue loop: start", tag)
		defer e.xlog().Infof("deliver(%s): queue loop: stop", tag)
		for evt := range queue {
			stateCtx := NewStatefulContext(ctx, StateAsync)
			for _, router := range e._pipeline {
				if err := e.route(stateCtx, router, evt); err != nil {
					e.xlog().Errorf("router.route error: %s", err)
				}
			}
		}
	}
	// start inputs
	e.xlog().Infof("ENGINE: START INPUTS, count: %d", len(e._inputs))
	for _, input := range e._inputs {
		// 每个Input维护独立的Queue
		go func(in Input) {
			queue := make(chan Event, e.queueSize)
			defer close(queue)
			go doqueue(e.stateContext, in.Tag(), queue)
			e.xlog().Infof("start input, tag: %s", in.Tag())
			in.OnReceived(e.stateContext, queue)
		}(input)
	}
	return nil
}

func (e *EventEngine) route(stateCtx StateContext, pipeline *Pipeline, data Event) error {
	next := FilterFunc(func(ctx StateContext, evt Event) (err error) {
		// Transform
		events := []Event{evt}
		for _, tf := range pipeline.transformers {
			events, err = tf.DoTransform(ctx, events)
			if err != nil {
				return err
			}
		}
		for _, output := range pipeline.outputs {
			output.OnSend(ctx.Context(), events...)
		}
		return nil
	})
	fc := makeFilterChain(next, pipeline.filters)
	return fc(stateCtx, data)
}

func (e *EventEngine) GetPipelines() []*Pipeline {
	copied := make([]*Pipeline, len(e._pipeline))
	copy(copied, e._pipeline)
	return copied
}

func (e *EventEngine) SetInputs(v []Input) {
	e._inputs = v
}

func (e *EventEngine) AddInput(v Input) {
	e.SetInputs(append(e._inputs, v))
}

func (e *EventEngine) SetOutputs(v []Output) {
	e._outputs = v
}

func (e *EventEngine) AddOutput(v Output) {
	e.SetOutputs(append(e._outputs, v))
}

func (e *EventEngine) SetFilters(v []Filter) {
	e._filters = v
}

func (e *EventEngine) AddFilter(v Filter) {
	e.SetFilters(append(e._filters, v))
}

func (e *EventEngine) SetTransformers(v []Transformer) {
	e._transformers = v
}

func (e *EventEngine) AddTransformer(v Transformer) {
	e.SetTransformers(append(e._transformers, v))
}

func (e *EventEngine) Order(state runv.State) int {
	return 10_0000 // 所有生命周期都靠后
}

func makeFilterChain(next FilterFunc, filters []Filter) FilterFunc {
	for i := len(filters) - 1; i >= 0; i-- {
		next = filters[i].DoFilter(next)
	}
	return next
}

func (e *EventEngine) compile(definitions []PipelineDefinition) {
	for _, definition := range definitions {
		assert.Must(definition.Description != "", "pipeline definition, 'description' is required")
		assert.Must(definition.Selector.Input != "", "pipeline definition, selector 'input-tags' is required")
		assert.Must(len(definition.Selector.Outputs) > 0, "pipeline definition, selector 'output-tags' is required")
		verify := func(tags []string, msg, src string) {
			for _, t := range tags {
				assert.Must(len(t) > 0, msg, t, src)
			}
		}
		for _, pd := range e.flat(definition) {
			assert.Must(len(pd.Input) > 0, "pipeline, 'input' tag is invalid, tag: "+pd.Input+", desc: "+definition.Description)
			verify(pd.Filters, "pipeline, 'filter' tag is invalid, tag: %s, src: %s", pd.Input)
			verify(pd.Transformers, "pipeline, 'transformer' tag is invalid, tag: %s, src: %s", pd.Input)
			verify(pd.Outputs, "pipeline, 'output' tag is invalid, tag: %s, src: %s", pd.Input)
			pipeline := NewPipeline(pd.Input)
			Log().Infof("ENGINE: BIND-PIPELINE, input: %s, pipeline: %+v", pd.Input, pd)
			e._pipeline = append(e._pipeline, e.lookup(pipeline, pd))
		}
	}
}

func (e *EventEngine) statechk() error {
	assert.Must(0 < len(e._pipeline), "engine.pipelines is required")
	return nil
}

func (e *EventEngine) flat(definition PipelineDefinition) []PipelineDescriptor {
	descriptors := make([]PipelineDescriptor, 0, len(e._inputs))
	newMatcher([]string{definition.Selector.Input}).on(e._inputs, func(v interface{}) {
		descriptors = append(descriptors, PipelineDescriptor{
			Description:  definition.Description,
			Input:        v.(Input).Tag(),
			Filters:      definition.Selector.Filter,
			Transformers: definition.Selector.Transformers,
			Outputs:      definition.Selector.Outputs,
		})
	})
	return descriptors
}

func (e *EventEngine) lookup(pipeline *Pipeline, descriptor PipelineDescriptor) *Pipeline {
	// filters
	newMatcher(descriptor.Filters).on(e._filters, func(v interface{}) {
		pipeline.AddFilter(v.(Filter))
	})
	// transformer
	newMatcher(descriptor.Transformers).on(e._transformers, func(v interface{}) {
		pipeline.AddTransformer(v.(Transformer))
	})
	// output
	newMatcher(descriptor.Outputs).on(e._outputs, func(v interface{}) {
		pipeline.AddOutput(v.(Output))
	})
	return pipeline
}

func (e *EventEngine) xlog() *logrus.Entry {
	return Log().WithField("app", "engine")
}

func WithQueueSize(size uint) EventEngineOption {
	if size == 0 {
		size = 10
	}
	return func(d *EventEngine) {
		d.queueSize = size
	}
}
