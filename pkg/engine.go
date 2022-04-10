package flow

import (
	"context"
	"fmt"
	"github.com/bytepowered/runv"
	"github.com/bytepowered/runv/assert"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
	"sync"
)

var (
	_ runv.Initable  = new(EventEngine)
	_ runv.Liveness  = new(EventEngine)
	_ runv.Liveorder = new(EventEngine)
	_ runv.Servable  = new(EventEngine)
)

const (
	WorkModeSingle   = "single"
	WorkModePipeline = "pipeline"
)

type EventEngineOption func(*EventEngine)

type EventEngine struct {
	_inputs       []Input
	_filters      []Filter
	_transformers []Transformer
	_outputs      []Output
	_pipelines    []*Pipeline
	queueSize     uint
	workMode      string
	stateContext  context.Context
	stateFunc     context.CancelFunc
}

const (
	engineConfigQueueSizeKey = "engine.queue-size"
	engineConfigOrderKey     = "engine.order"
)

func NewEventEngine(opts ...EventEngineOption) *EventEngine {
	viper.SetDefault(engineConfigQueueSizeKey, 10)
	viper.SetDefault(engineConfigOrderKey, 10000)
	ctx, ctxfunc := context.WithCancel(context.Background())
	engine := &EventEngine{
		stateContext: ctx, stateFunc: ctxfunc,
		queueSize: viper.GetUint(engineConfigQueueSizeKey),
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
	// WorkMode: 决定Engine对事件流的处理方式
	// 1. Single: 所有组件组合为单个事件处理链；
	// 2. Pipeline：根据Pipeline定义来定义多个事件处理链；
	viper.SetDefault("engine.workmode", WorkModePipeline)
	mode := viper.GetString("engine.workmode")
	switch strings.ToLower(mode) {
	case WorkModeSingle:
		groups = append(groups, PipelineDefinition{
			Description: "run on single mode",
			Selector: PipelineSelector{
				InputExpr:        "*",
				FiltersExpr:      []string{"*"},
				TransformersExpr: []string{"*"},
				OutputsExpr:      []string{"*"},
			},
		})
	default:
		fallthrough
	case WorkModePipeline:
		if err := UnmarshalConfigKey("pipeline", &groups); err != nil {
			return fmt.Errorf("load 'routers' config error: %w", err)
		}
	}
	e.compile(groups)
	e.xlog().Infof("ENGINE: INIT, WorkMode: %s, pipelines: %d", mode, len(groups))
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
	e.xlog().Infof("ENGINE: SERVE, start, input-count: %d", len(e._inputs))
	defer e.xlog().Infof("ENGINE: SERVE, stop")
	inputwg := new(sync.WaitGroup)
	for _, input := range e._inputs {
		binds := make([]*Pipeline, 0, len(e._pipelines))
		for _, p := range e._pipelines {
			if input.Tag() == p.Input {
				binds = append(binds, p)
			}
		}
		// 确保每个Input至少绑定一个Pipeline
		if len(binds) == 0 {
			e.xlog().Infof("ENGINE: SKIP-INPUT, NO PIPELINES, tag: %s", input.Tag())
			continue
		}
		// 基于输入源Input来启动独立协程
		inputwg.Add(1)
		go func(in Input, binds []*Pipeline) {
			defer inputwg.Done()
			e.xlog().Infof("ENGINE: START-INPUT, tag: %s", in.Tag())
			queue := make(chan Event, e.queueSize)
			qdone := make(chan struct{}, 0)
			go func(qdone chan struct{}) {
				defer close(qdone)
				e.xlog().Infof("ENGINE: INPUT-QUEUE-START: tag: %s", in.Tag())
				defer e.xlog().Infof("ENGINE: INPUT-QUEUE-STOP: tag: %s", in.Tag())
				e.qconsume(e.stateContext, in.Tag(), binds, queue)
			}(qdone)
			// 确保关闭缓存队列
			func() {
				defer close(queue)
				in.OnReceived(e.stateContext, queue)
			}()
			<-qdone // 等待消费队列完成
		}(input, binds)
	}
	inputwg.Wait()
	return nil
}

func (e *EventEngine) qconsume(ctx context.Context, tag string, pipelines []*Pipeline, queue <-chan Event) {
	for evt := range queue {
		stateCtx := NewStatefulContext(ctx, StateAsync)
		for _, bind := range pipelines {
			if err := e.route(stateCtx, bind, evt); err != nil {
				e.xlog().Errorf("pipeline.route error, input: %s, error: %s", tag, err)
			}
		}
	}
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
	copied := make([]*Pipeline, len(e._pipelines))
	copy(copied, e._pipelines)
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
	return viper.GetInt(engineConfigOrderKey) // 所有生命周期都靠后
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
		assert.Must(definition.Selector.InputExpr != "", "pipeline definition, selector 'input-tags' is required")
		assert.Must(len(definition.Selector.OutputsExpr) > 0, "pipeline definition, selector 'output-tags' is required")
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
			e._pipelines = append(e._pipelines, e.register(pipeline, pd))
		}
	}
}

func (e *EventEngine) statechk() error {
	assert.Must(0 < len(e._pipelines), "engine.pipelines is required")
	return nil
}

func (e *EventEngine) flat(definition PipelineDefinition) []PipelineDescriptor {
	descriptors := make([]PipelineDescriptor, 0, len(e._inputs))
	// 从Input实例列表中，根据Tag匹配实例对象
	newMatcher([]string{definition.Selector.InputExpr}).on(e._inputs, func(tag string, _ interface{}) {
		descriptors = append(descriptors, PipelineDescriptor{
			Description:  definition.Description,
			Input:        tag,
			Filters:      definition.Selector.FiltersExpr,
			Transformers: definition.Selector.TransformersExpr,
			Outputs:      definition.Selector.OutputsExpr,
		})
	})
	return descriptors
}

func (e *EventEngine) register(pipeline *Pipeline, descriptor PipelineDescriptor) *Pipeline {
	// filters
	newMatcher(descriptor.Filters).on(e._filters, func(tag string, v interface{}) {
		pipeline.AddFilter(v.(Filter))
	})
	// transformer
	newMatcher(descriptor.Transformers).on(e._transformers, func(tag string, v interface{}) {
		pipeline.AddTransformer(v.(Transformer))
	})
	// output
	newMatcher(descriptor.Outputs).on(e._outputs, func(tag string, v interface{}) {
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
