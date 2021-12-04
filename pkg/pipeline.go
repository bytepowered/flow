package flow

import "fmt"

var _ EventEmitter = new(Pipeline)

type PipelineEmitFunc func(*Pipeline, EventContext, EventRecord)

type Pipeline struct {
	workf        PipelineEmitFunc
	filters      []EventFilter
	transformers []Transformer
	dispatchers  []Dispatcher
}

func NewPipeline() *Pipeline {
	return NewPipelineOf(nil)
}

func NewPipelineOf(workf PipelineEmitFunc) *Pipeline {
	return &Pipeline{
		workf:        workf,
		filters:      make([]EventFilter, 0, 2),
		transformers: make([]Transformer, 0, 2),
		dispatchers:  make([]Dispatcher, 0, 2),
	}
}

func (p *Pipeline) AddEventFilter(f EventFilter) {
	p.filters = append(p.filters, f)
}

func (p *Pipeline) AddTransformer(t Transformer) {
	p.transformers = append(p.transformers, t)
}

func (p *Pipeline) AddDispatcher(d Dispatcher) {
	p.dispatchers = append(p.dispatchers, d)
}

func (p *Pipeline) Emit(context EventContext, record EventRecord) {
	p.workf(p, context, record)
}

func (p *Pipeline) doEmit(context EventContext, record EventRecord) {
	// filter -> transformer -> dispatcher
	next := EventFilterFunc(func(ctx EventContext, record EventRecord) (err error) {
		for _, tf := range p.transformers {
			record, err = tf.DoTransform(record)
			if err != nil {
				return err
			}
		}
		for _, df := range p.dispatchers {
			if err = df.DoDelivery(record); err != nil {
				return fmt.Errorf("dispatch(%s) error: %w", df.Tag(), err)
			}
		}
		return nil
	})
	if err := p.makeFilterChain(next, p.filters)(context, record); err != nil {
		Log().Errorf("pipeline handle event, error: %s", err)
	}
}

func (p *Pipeline) makeFilterChain(next EventFilterFunc, filters []EventFilter) EventFilterFunc {
	for i := len(filters) - 1; i >= 0; i-- {
		next = filters[i].DoFilter(next)
	}
	return next
}
