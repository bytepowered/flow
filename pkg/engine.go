package flow

import (
	"context"
	"fmt"
	"github.com/bytepowered/flow/v3/pkg/compat"
	"github.com/bytepowered/runv"
	"github.com/sirupsen/logrus"
	"sync"
)

var (
	_ runv.Initable  = new(EventEngine)
	_ runv.Liveness  = new(EventEngine)
	_ runv.Liveorder = new(EventEngine)
)

type EventEngineOption func(*EventEngine)

type EventEngine struct {
	_inputs       []Input
	_filters      []Filter
	_transformers []Transformer
	_outputs      []Output
	_routers      []*Router
	queueSize     uint
	stateContext  context.Context
	stateFunc     context.CancelFunc
}

func NewEventEngine(opts ...EventEngineOption) *EventEngine {
	cc, cf := context.WithCancel(context.Background())
	engine := &EventEngine{
		stateContext: cc, stateFunc: cf,
		queueSize: 10,
	}
	for _, opt := range opts {
		opt(engine)
	}
	runv.AddPostHook(engine.statechk)
	return engine
}

func (e *EventEngine) OnInit() error {
	e.xlog().Infof("init")
	runv.Assert(0 < len(e._inputs), "engine.inputs is required")
	runv.Assert(0 < len(e._outputs), "engine.outputs is required")
	// 从配置文件中加载route配置项
	groups := make([]GroupRouter, 0)
	if err := UnmarshalConfigKey("router", &groups); err != nil {
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
	e.stateFunc()
	e.xlog().Infof("shutdown")
	return nil
}

func (e *EventEngine) Serve() {
	deliver := func(ctx context.Context, tag string, queue chan<- Event) {
		e.xlog().Infof("deliver(%s): queue loop: start", tag)
		defer e.xlog().Infof("deliver(%s): queue loop: stop", tag)
		stateCtx := compat.NewStatedContext(ctx, StateAsync)
		for evt := range queue {
			for _, pipe := range e._routers {
				err := e.deliver(stateCtx, pipe, evt)
				if err != nil {
					e.xlog().Errorf("router deliver error: %s", err)
				}
			}
		}
	}
	// start inputs
	wg := new(sync.WaitGroup)
	e.xlog().Infof("start inputs, count: %d", len(e._inputs))
	for _, input := range e._inputs {
		wg.Add(1)
		// 每个Input维护独立的Queue
		go func(in Input) {
			defer wg.Done()
			queue := make(chan Event, e.queueSize)
			defer close(queue)
			go deliver(e.stateContext, in.Tag(), queue)
			e.xlog().Infof("start input, tag: %s", in.Tag())
			in.OnReceive(e.stateContext, queue)
		}(input)
	}
	wg.Wait()
}

func (e *EventEngine) deliver(ctx StateContext, pipe *Router, event Event) error {
	next := FilterFunc(func(ctx StateContext, evt Event) (err error) {
		// Transform
		for _, tf := range pipe.transformers {
			evt, err = tf.DoTransform(evt)
			if err != nil {
				return err
			}
		}
		for _, output := range pipe.outputs {
			output.OnSend(ctx.Context(), evt)
		}
		return nil
	})
	fc := e.makeFilterChain(next, pipe.filters)
	return fc(ctx, event)
}

func (e *EventEngine) statechk() error {
	runv.Assert(0 < len(e._routers), "engine.routers is required")
	return nil
}

func (e *EventEngine) SetInputs(v []Input) {
	e._inputs = v
}

func (e *EventEngine) SetOutputs(v []Output) {
	e._outputs = v
}

func (e *EventEngine) SetFilters(v []Filter) {
	e._filters = v
}

func (e *EventEngine) SetTransformers(v []Transformer) {
	e._transformers = v
}

func (e *EventEngine) Order(state runv.State) int {
	return 10_0000 // 所有生命周期都靠后
}

func (e *EventEngine) makeFilterChain(next FilterFunc, filters []Filter) FilterFunc {
	for i := len(filters) - 1; i >= 0; i-- {
		next = filters[i].DoFilter(next)
	}
	return next
}

func (e *EventEngine) compile(groups []GroupRouter) {
	for _, group := range groups {
		runv.Assert(group.Description != "", "router group, 'description' is required")
		runv.Assert(len(group.Selector.InputTags) > 0, "router group, selector 'input-tags' is required")
		runv.Assert(len(group.Selector.OutputTags) > 0, "router group, selector 'output-tags' is required")
		verify := func(tags []string, msg, src string) {
			for _, t := range tags {
				runv.Assert(len(t) >= 3, msg, t, src)
			}
		}
		for _, tr := range e.flat(group) {
			runv.Assert(len(tr.InputTag) >= 3, "router, 'input-tag' is invalid, tag: "+tr.InputTag+", group: "+group.Description)
			verify(tr.FilterTags, "router, 'filter-tag' is invalid, tag: %s, src: %s", tr.InputTag)
			verify(tr.TransformerTags, "router, 'transformer-tag' is invalid, tag: %s, src: %s", tr.InputTag)
			verify(tr.OutputTags, "router, 'output-tag' is invalid, tag: %s, src: %s", tr.InputTag)
			Log().Infof("bind router, src.tag: %s, route: %+v", tr.InputTag, tr)
			router := NewRouter()
			e._routers = append(e._routers, e.lookup(router, tr))
		}
	}
}

func (e *EventEngine) flat(group GroupRouter) []TaggedRouter {
	routers := make([]TaggedRouter, 0, len(e._inputs))
	TagMatcher(group.Selector.InputTags).match(e._inputs, func(v interface{}) {
		routers = append(routers, TaggedRouter{
			description:     group.Description,
			InputTag:        v.(Input).Tag(),
			FilterTags:      group.Selector.FilterTags,
			TransformerTags: group.Selector.TransformerTags,
			OutputTags:      group.Selector.OutputTags,
		})
	})
	return routers
}

func (e *EventEngine) lookup(router *Router, tags TaggedRouter) *Router {
	// filters
	TagMatcher(tags.FilterTags).match(e._filters, func(v interface{}) {
		router.AddFilter(v.(Filter))
	})
	// transformer
	TagMatcher(tags.TransformerTags).match(e._transformers, func(v interface{}) {
		router.AddTransformer(v.(Transformer))
	})
	// output
	TagMatcher(tags.OutputTags).match(e._outputs, func(v interface{}) {
		router.AddOutput(v.(Output))
	})
	return router
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
