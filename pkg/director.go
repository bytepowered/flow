package flow

import (
	"context"
	"fmt"
	"github.com/Jeffail/tunny"
	"github.com/bytepowered/runv"
)

var (
	_ EventDeliverFunc = new(EventDirector)
	_ runv.Initable    = new(EventDirector)
	_ runv.Shutdown    = new(EventDirector)
)

type DirectorOption func(director *EventDirector)

type EventDirector struct {
	workers       *tunny.Pool
	dispatchers   map[string]Dispatcher
	adapters      map[string]Adapter
	globalFilters []EventFilter
	effectFilters map[EventType][]EffectEventFilter
	transformers  map[Vendor]Transformer
}

func NewEventDirector(opts ...DirectorOption) *EventDirector {
	fd := &EventDirector{
		dispatchers:   make(map[string]Dispatcher, 2),
		adapters:      make(map[string]Adapter, 2),
		globalFilters: make([]EventFilter, 0, 4),
		effectFilters: make(map[EventType][]EffectEventFilter, 2),
		transformers:  make(map[Vendor]Transformer, 2),
	}
	fd.workers = tunny.NewFunc(10_000, func(i interface{}) interface{} {
		args := i.([]interface{})
		header, frame := args[0].(EventHeader), args[1].([]byte)
		return fd.work(header, frame)
	})
	for _, opt := range opts {
		opt(fd)
	}
	return fd
}

func (d *EventDirector) OnInit() error {
	runv.Assert(0 < len(d.transformers), "transformers is required")
	runv.Assert(0 < len(d.adapters), "adapters is required")
	runv.Assert(0 < len(d.dispatchers), "dispatchers is required")
	for _, adapter := range d.adapters {
		adapter.SetEventDeliverFunc(d)
	}
	return nil
}

func (d *EventDirector) Deliver(ctx EventContext, header EventHeader, packet []byte) {
	defer func() {
		if err := recover(); err != nil {
			Log().Errorf("unexcepted panic error: %s", err)
		}
	}()
	metrics := Metrics()
	const mstage = "director"
	mroute := metrics.NewTimer(mstage, "route")
	defer mroute.ObserveDuration()
	source := header.Origin.String() + "_" + header.Vendor.String()
	metrics.NewCounter(source, header.EventType.String()).Inc()
	// select filter
	effective := d.effectFilters[header.EventType]
	filters := make([]EventFilter, len(d.globalFilters), len(d.globalFilters)+len(effective))
	copy(filters, d.globalFilters)
	for _, f := range effective {
		filters = append(filters, f)
	}
	// do filter
	mfilter := metrics.NewTimer(mstage, "filter")
	_ = d.makeFilterChain(func(eh EventHeader) (err error) {
		mfilter.ObserveDuration()
		if ctx.Async() {
			err = d.work(eh, packet)
		} else {
			if e := d.workers.Process([]interface{}{eh, packet}); e != nil {
				err = e.(error)
			}
		}
		if err != nil {
			Log().Errorf("handle/filter event, error: %s", err)
		}
		return nil
	}, filters)(header)
}

func (d *EventDirector) Shutdown(ctx context.Context) error {
	d.workers.Close()
	return nil
}

func (d *EventDirector) work(header EventHeader, packet []byte) error {
	// transform
	transformer, ok := d.transformers[header.Vendor]
	if !ok {
		return fmt.Errorf("unsupported transformer, vendor: %s", VendorName(header.Vendor))
	}
	event, err := transformer.DoTransform(header, packet)
	if err != nil {
		return fmt.Errorf("failed to format event, error: %s", err)
	}
	// dispatch
	for _, dispatch := range d.dispatchers {
		if err := dispatch.Dispatch(event); err != nil {
			return fmt.Errorf("dispatcher[%T] dispatch error: %s", dispatch, err)
		}
	}
	return nil
}

func (d *EventDirector) makeFilterChain(next EventFilterFunc, filters []EventFilter) EventFilterFunc {
	for i := len(filters) - 1; i >= 0; i-- {
		next = filters[i].DoFilter(next)
	}
	return next
}

func (d *EventDirector) SetTransformers(formatters []Transformer) {
	Log().Infof("director set transformers: %d", len(formatters))
	for _, f := range formatters {
		d.transformers[f.ActiveON()] = f
	}
}

func (d *EventDirector) SetGlobalFilters(filters []EventFilter) {
	Log().Infof("director add global filters: %d", len(filters))
	for _, f := range filters {
		// 忽略Effective EffectEventFilter
		if _, is := f.(EffectEventFilter); is {
			continue
		}
		d.globalFilters = append(d.globalFilters, f)
	}
}

func (d *EventDirector) SetEffectFilters(filters []EffectEventFilter) {
	Log().Infof("director add effect filters: %d", len(filters))
	for _, filter := range filters {
		if list, ok := d.effectFilters[filter.EffectON()]; ok {
			d.effectFilters[filter.EffectON()] = append(list, filter)
		} else {
			d.effectFilters[filter.EffectON()] = []EffectEventFilter{filter}
		}
	}
}

func (d *EventDirector) SetDispatchers(items []Dispatcher) {
	Log().Infof("director add dispatchers: %d", len(items))
	for _, a := range items {
		d.dispatchers[a.DispatcherId()] = a
	}
}

func (d *EventDirector) SetAdapters(items []Adapter) {
	Log().Infof("director add adapters: %d", len(items))
	for _, a := range items {
		d.adapters[a.AdapterId()] = a
	}
}

func WithFlowDirectorWorkerSize(size uint) DirectorOption {
	return func(d *EventDirector) {
		d.workers.SetSize(int(size))
	}
}
