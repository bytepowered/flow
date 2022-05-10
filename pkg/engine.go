package flow

import (
	"context"
	"fmt"
	"github.com/bytepowered/runv"
	"github.com/bytepowered/runv/assert"
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
	_pipelines    []Pipeline
	queueSize     uint
	workMode      string
	stateContext  context.Context
	stateFunc     context.CancelFunc
}

const (
	engineConfigQueueSizeKey = "engine.queue_size"
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
	Log().Infof("ENGINE: INIT")
	assert.Must(0 < len(e._inputs), "engine.inputs is required")
	assert.Must(0 < len(e._outputs), "engine.outputs is required")
	// 从配置文件中加载route配置项
	groups := make([]Definition, 0)
	// WorkMode: 决定Engine对事件流的处理方式
	// 1. Single: 所有组件组合为单个事件处理链；
	// 2. Pipeline：根据Pipeline定义来定义多个事件处理链；
	viper.SetDefault("engine.workmode", WorkModePipeline)
	mode := viper.GetString("engine.workmode")
	switch strings.ToLower(mode) {
	case WorkModeSingle:
		groups = append(groups, Definition{
			Description: "run on single mode",
			Selector: TagSelector{
				Input:        "*",
				Filters:      []string{"*"},
				Transformers: []string{"*"},
				Outputs:      []string{"*"},
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
	Log().Infof("ENGINE: INIT, WorkMode: %s, pipelines: %d", mode, len(groups))
	return nil
}

func (e *EventEngine) Startup(ctx context.Context) error {
	Log().Infof("ENGINE: STARTUP")
	return nil
}

func (e *EventEngine) Shutdown(ctx context.Context) error {
	e.stateFunc()
	Log().Infof("ENGINE: SHUTDOWN")
	return nil
}

func (e *EventEngine) Serve(c context.Context) error {
	Log().Infof("ENGINE: SERVE, start, input-count: %d", len(e._inputs))
	defer Log().Infof("ENGINE: SERVE, stop")
	inputwg := new(sync.WaitGroup)
	// MARK 基于Input而非Pipeline，每个Input可被多个Pipeline绑定
	for _, input := range e._inputs {
		binds := make([]Pipeline, 0, len(e._pipelines))
		for _, p := range e._pipelines {
			if input.Tag() == p.Input {
				binds = append(binds, p)
			}
		}
		// 确保每个Input至少绑定一个Pipeline
		if len(binds) == 0 {
			Log().Warnf("ENGINE: SKIP-INPUT, NO PIPELINES, tag: %s", input.Tag())
			continue
		}
		// 基于输入源Input来启动独立协程
		inputwg.Add(1)
		go func(in Input, binds []Pipeline) {
			defer inputwg.Done()
			Log().Infof("ENGINE: START-INPUT: %s", in.Tag())
			qsize := e.queueSize
			// Input configure
			if ref, ok0 := in.(Configurable); ok0 {
				if qc, ok1 := ref.OnConfigure().(InputConfig); ok1 {
					qsize = qc.QueueSize
				}
			}
			queue := make(chan Event, qsize)
			qdone := make(chan struct{}, 0)
			go func(qdone chan struct{}) {
				defer close(qdone)
				Log().Infof("ENGINE: INPUT-QUEUE-START: %s", in.Tag())
				defer Log().Infof("ENGINE: INPUT-QUEUE-STOP: %s", in.Tag())
				e.queueconsume(e.stateContext, in.Tag(), binds, queue)
			}(qdone)
			defer func() {
				if r := recover(); r != nil {
					panic(fmt.Errorf("ENGINE: INPUT-PANIC(%s): %+v", in.Tag(), r))
				}
			}()
			// FuncCall: 确保关闭缓存队列
			func() {
				defer close(queue)
				in.OnRead(e.stateContext, queue)
			}()
			<-qdone // 等待消费队列完成
		}(input, binds)
	}
	inputwg.Wait()
	return nil
}

func (e *EventEngine) queueconsume(ctx context.Context, tag string, pipelines []Pipeline, queue <-chan Event) {
	for evt := range queue {
		stateCtx := NewStatefulContext(ctx, StateAsync)
		for _, bind := range pipelines {
			if err := e.dispatch(stateCtx, bind, evt); err != nil {
				Log().Errorf("ENGINE: QUEUE-CONSUME-ERROR, input: %s, error: %s", tag, err)
			}
		}
	}
}

func (e *EventEngine) dispatch(stateCtx StateContext, pipeline Pipeline, data Event) error {
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

func (e *EventEngine) GetPipelines() []Pipeline {
	copied := make([]Pipeline, len(e._pipelines))
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

func (e *EventEngine) compile(definitions []Definition) {
	for _, definition := range definitions {
		assert.Must(definition.Description != "", "pipeline definition, 'description' is required")
		assert.Must(definition.Selector.Input != "", "pipeline definition, selector 'input-tags' is required")
		assert.Must(len(definition.Selector.Outputs) > 0, "pipeline definition, selector 'output-tags' is required")
		verify := func(targets []string, msg, src string) {
			for _, target := range targets {
				assert.Must(len(target) > 0, msg, target, src)
			}
		}
		for _, df := range e.flat(definition) {
			tag := df.Selector.Input
			assert.Must(len(df.Selector.Input) > 0, "pipeline, 'input' tag is invalid, tag: "+tag+", desc: "+df.Description)
			verify(df.Selector.Filters, "pipeline, 'filter' tag is invalid, tag: %s, src: %s", tag)
			verify(df.Selector.Transformers, "pipeline, 'transformer' tag is invalid, tag: %s, src: %s", tag)
			verify(df.Selector.Outputs, "pipeline, 'output' tag is invalid, tag: %s, src: %s", tag)
			pipeline := NewPipeline(tag)
			Log().Infof("ENGINE: BIND-PIPELINE, input: %s, pipeline: %+v", tag, df)
			e._pipelines = append(e._pipelines, e.register(pipeline, df))
		}
	}
}

func (e *EventEngine) statechk() error {
	assert.Must(0 < len(e._pipelines), "engine.pipelines is required")
	return nil
}

func (e *EventEngine) flat(define Definition) []Definition {
	definitions := make([]Definition, 0, len(e._inputs))
	// 从Input实例列表中，根据Tag匹配实例对象
	newMatcher([]string{define.Selector.Input}).on(e._inputs, func(tag string, _ interface{}) {
		selector := define.Selector
		selector.Input = tag // Update tag
		definitions = append(definitions, Definition{
			Description: define.Description,
			Selector:    selector,
		})
	})
	return definitions
}

func (e *EventEngine) register(pipeline *Pipeline, define Definition) Pipeline {
	// filters
	newMatcher(define.Selector.Filters).on(e._filters, func(tag string, v interface{}) {
		pipeline.AddFilter(v.(Filter))
	})
	// transformer
	newMatcher(define.Selector.Transformers).on(e._transformers, func(tag string, v interface{}) {
		pipeline.AddTransformer(v.(Transformer))
	})
	// output
	newMatcher(define.Selector.Outputs).on(e._outputs, func(tag string, v interface{}) {
		pipeline.AddOutput(v.(Output))
	})
	return *pipeline
}

func WithQueueSize(size uint) EventEngineOption {
	if size == 0 {
		size = 10
	}
	return func(d *EventEngine) {
		d.queueSize = size
	}
}
