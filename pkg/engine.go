package flow

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
	"sync"
)

var (
	_ Initable = new(EventEngine)
	_ Servable = new(EventEngine)
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
	stateCtx      context.Context
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
		stateCtx: ctx, stateFunc: ctxfunc,
		queueSize: viper.GetUint(engineConfigQueueSizeKey),
	}
	for _, opt := range opts {
		opt(engine)
	}
	return engine
}

func (e *EventEngine) OnInit() error {
	Log().Infof("ENGINE: INIT")
	Must(0 < len(e._inputs), "engine.inputs is required")
	Must(0 < len(e._outputs), "engine.outputs is required")
	// 从配置文件中加载route配置项
	groups := make([]Definition, 0)
	// WorkMode: 决定Engine对事件流的处理方式
	// 1. Single: 所有组件组合为单个事件处理链；
	// 2. Pipeline：根据Pipeline定义来定义多个事件处理链；
	viper.SetDefault("engine.workmode", WorkModePipeline)
	mode := viper.GetString("engine.workmode")
	switch strings.ToLower(mode) {
	case WorkModeSingle:
		wildcard := []string{"*"}
		groups = append(groups, Definition{
			Description: "run on single mode",
			Selector: TagSelector{
				Input: "*", Filters: wildcard, Transformers: wildcard, Outputs: wildcard,
			},
		})
	default:
		fallthrough
	case WorkModePipeline:
		if err := UnmarshalConfigKey("pipeline", &groups); err != nil {
			return fmt.Errorf("load 'pipeline' config error: %w", err)
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

func (e *EventEngine) Serve(_ context.Context) error {
	var verbose logrus.Level
	if viper.GetBool("verbose") {
		verbose = logrus.InfoLevel
	} else {
		verbose = logrus.DebugLevel
	}
	Log().Infof("ENGINE: SERVE, start, input: %d, output: %d", len(e._inputs), len(e._outputs))
	defer Log().Infof("ENGINE: SERVE, stop")
	wg := new(sync.WaitGroup)
	// 每个Input可被多个Pipeline绑定
	for _, input := range e._inputs {
		e.start(input, wg, verbose)
	}
	wg.Wait()
	return nil
}

func (e *EventEngine) Pipelines() []Pipeline {
	copied := make([]Pipeline, len(e._pipelines))
	copy(copied, e._pipelines)
	return copied
}

func (e *EventEngine) AddPipeline(pipeline Pipeline) {
	e._pipelines = append(e._pipelines, pipeline)
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

func (e *EventEngine) start(input Input, inputswg *sync.WaitGroup, verbose logrus.Level) {
	find := func(tag string) []Pipeline {
		out := make([]Pipeline, 0, len(e._pipelines))
		for _, p := range e._pipelines {
			if tag == p.Input {
				out = append(out, p)
			}
		}
		return out
	}
	binds := find(input.Tag())
	// 确保每个Input至少绑定一个Pipeline
	if len(binds) == 0 {
		Log().Warnf("ENGINE: SKIP-INPUT, NO PIPELINES, tag: %s", input.Tag())
		return
	}
	go func(in Input) {
		defer func() {
			Log().Infof("ENGINE: READ-LOOP-STOP: %s", in.Tag())
			if r := recover(); r != nil {
				panic(fmt.Errorf("ENGINE: READ-LOOP-PANIC(%s): %+v", in.Tag(), r))
			}
		}()
		Log().Infof("ENGINE: READ-LOOP-START: %s", in.Tag())
		in.OnRead(e.stateCtx, func(inbound Event) {
			Must(in.Tag() == inbound.Tag(), "ENGINE: BAD-EVENT-TAG: %s -> %s", in.Tag(), inbound.Tag())
			for _, pipeline := range e._pipelines {
				pipeline.worker(e.wrap(pipeline, inbound))
			}
		})
	}(input)
}

func (e *EventEngine) wrap(pipeline Pipeline, inbound Event) func() {
	return func() {
		// Filter -> Transform -> Output
		err := makeFilterChain(func(ctx context.Context, in Event) (err error) {
			events := []Event{in}
			for _, tf := range pipeline.transformers {
				events, err = tf.DoTransform(ctx, events)
				if err != nil {
					return err
				}
			}
			for _, output := range pipeline.outputs {
				output.OnSend(ctx, events...)
			}
			return nil
		}, pipeline.filters)(e.stateCtx, inbound)
		if err != nil {
			Log().Errorf("ENGINE: DISPATCH-ERROR, input: %s, error: %s", inbound.Tag(), err)
		}
	}
}

func (e *EventEngine) compile(definitions []Definition) {
	for _, definition := range definitions {
		Must(definition.Description != "", "pipeline definition, 'description' is required")
		Must(definition.Selector.Input != "", "pipeline definition, selector 'input-tags' is required")
		Must(len(definition.Selector.Outputs) > 0, "pipeline definition, selector 'output-tags' is required")
		verify := func(targets []string, msg, src string) {
			for _, target := range targets {
				Must(len(target) > 0, msg, target, src)
			}
		}
		for _, df := range e.flat(definition) {
			tag := df.Selector.Input
			Must(len(df.Selector.Input) > 0, "pipeline, 'input' tag is invalid, tag: "+tag+", desc: "+df.Description)
			verify(df.Selector.Filters, "pipeline, 'filter' tag is invalid, tag: %s, src: %s", tag)
			verify(df.Selector.Transformers, "pipeline, 'transformer' tag is invalid, tag: %s, src: %s", tag)
			verify(df.Selector.Outputs, "pipeline, 'output' tag is invalid, tag: %s, src: %s", tag)
			pipeline := NewPipeline(tag)
			Log().Infof("ENGINE: BIND-PIPELINE, input: %s, pipeline: %+v", tag, df)
			e.AddPipeline(e.initialize(pipeline, df))
		}
	}
}

func (e *EventEngine) flat(define Definition) []Definition {
	definitions := make([]Definition, 0, len(e._inputs))
	// 从Input实例列表中，根据Tag匹配实例对象
	matches([]string{define.Selector.Input}, e._inputs, func(tag string, in Input) {
		selector := define.Selector
		selector.Input = tag // Update tag
		definitions = append(definitions, Definition{
			Description: define.Description,
			Selector:    selector,
		})
	}, func(tag string) {
		panic(fmt.Errorf("engine.pipeline definition.input(%s) not found", tag))
	})
	return definitions
}

func (e *EventEngine) initialize(pipeline *Pipeline, define Definition) Pipeline {
	// filters
	lookup(define.Selector.Filters, e._filters, func(tag string, plg Filter) {
		pipeline.AddFilter(plg)
	})
	// transformer
	lookup(define.Selector.Transformers, e._transformers, func(tag string, plg Transformer) {
		pipeline.AddTransformer(plg)
	})
	// output
	lookup(define.Selector.Outputs, e._outputs, func(tag string, plg Output) {
		pipeline.AddOutput(plg)
	})
	return *pipeline
}

func makeFilterChain(next FilterFunc, filters []Filter) FilterFunc {
	for i := len(filters) - 1; i >= 0; i-- {
		next = filters[i].DoFilter(next)
	}
	return next
}

func WithQueueSize(size uint) EventEngineOption {
	if size == 0 {
		size = 10
	}
	return func(d *EventEngine) {
		d.queueSize = size
	}
}
