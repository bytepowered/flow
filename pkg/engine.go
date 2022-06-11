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
	_mappings     map[string]Output
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
	// 启动时初始化Output的映射
	e._mappings = make(map[string]Output, len(e._outputs))
	for _, output := range e._outputs {
		e._mappings[output.Tag()] = output
	}
	getbinds := func(tag string) []Pipeline {
		out := make([]Pipeline, 0, len(e._pipelines))
		for _, p := range e._pipelines {
			if tag == p.Input {
				out = append(out, p)
			}
		}
		return out
	}
	Log().Infof("ENGINE: SERVE, start, input: %d, output: %d", len(e._inputs), len(e._outputs))
	defer Log().Infof("ENGINE: SERVE, stop")
	inputswg := new(sync.WaitGroup)
	// MARK 基于Input而非Pipeline，每个Input可被多个Pipeline绑定
	for _, input := range e._inputs {
		binds := getbinds(input.Tag())
		// 确保每个Input至少绑定一个Pipeline
		if len(binds) == 0 {
			Log().Warnf("ENGINE: SKIP-INPUT, NO PIPELINES, tag: %s", input.Tag())
			continue
		}
		qsize := e.queueSize
		if configer, ok := input.(Configurable); ok {
			if config, is := configer.OnConfigure().(InputConfig); is {
				qsize = config.QueueSize
			}
		}
		Log().Logf(verbose, "ENGINE: START-INPUT: %s, queue: %d, binds: %d", input.Tag(), qsize, len(binds))
		qevents := make(chan Event, qsize)
		// GO1: Input consume queue loop
		go func(in Input, pipelines []Pipeline, inqueue <-chan Event) {
			defer func() {
				Log().Logf(verbose, "ENGINE: INPUT-LOOP-STOP: %s", in.Tag())
				if r := recover(); r != nil {
					panic(fmt.Errorf("ENGINE: INPUT-LOOP-PANIC(%s): %+v", in.Tag(), r))
				}
			}()
			Log().Logf(verbose, "ENGINE: INPUT-LOOP-START: %s", in.Tag())
			e.qloop(e.stateCtx, in.Tag(), pipelines, inqueue)
		}(input, binds, qevents)
		// GO2: Input read loop
		inputswg.Add(1)
		go func(in Input, inqueue chan<- Event) {
			defer inputswg.Done()
			defer close(qevents)
			in.OnRead(e.stateCtx, inqueue)
		}(input, qevents)
	}
	inputswg.Wait()
	return nil
}

func (e *EventEngine) qloop(stateCtx context.Context, tag string, pipelines []Pipeline, events <-chan Event) {
	// 消费所有队列内的事件
	for recv := range events {
		Must(tag == recv.Tag(), "ENGINE: BAD-EVENT-TAG: %s -> %s", tag, recv.Tag())
		for _, pipe := range pipelines {
			err := e.dispatch(stateCtx, pipe, recv)
			if err != nil {
				Log().Errorf("ENGINE: DISPATCH-ERROR, input: %s, error: %s", tag, err)
			}
		}
	}
}

func (e *EventEngine) dispatch(evtctx context.Context, pipeline Pipeline, data Event) error {
	next := FilterFunc(func(ctx context.Context, src Event) (err error) {
		// 事件变换
		events := []Event{src}
		for _, tf := range pipeline.transformers {
			events, err = tf.DoTransform(ctx, events)
			if err != nil {
				return err
			}
		}
		// S1: 由Pipeline定义的Output处理
		for _, output := range pipeline.outputs {
			output.OnSend(ctx, events...)
		}
		// S2: 如果事件的Tag变更，投递到目标Output
		for _, evt := range events {
			output, ok := e._mappings[evt.Tag()]
			if ok {
				output.OnSend(ctx, evt)
			} else {
				Log().Warnf("ENGINE: DEAD-EVENT, input: %s, to: %s", src.Tag(), evt.Tag())
			}
		}
		return nil
	})
	fc := makeFilterChain(next, pipeline.filters)
	return fc(evtctx, data)
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
