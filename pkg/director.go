package flow

import (
	"context"
	"fmt"
	"github.com/Jeffail/tunny"
	"github.com/bytepowered/runv"
)

var (
	_ EventHandler   = new(EventDirector)
	_ runv.Initable  = new(EventDirector)
	_ runv.Component = new(EventDirector)
)

type DirectorOption func(director *EventDirector)

type EventDirector struct {
	workers       *tunny.Pool
	dispatchers   map[string]Dispatcher
	adapters      map[string]Adapter
	globalFilters []EventFilter
	effectFilters map[EventType][]TypedEventFilter
	transformers  map[Vendor]Transformer
}

func NewEventDirector(opts ...DirectorOption) *EventDirector {
	fd := &EventDirector{
		dispatchers:   make(map[string]Dispatcher, 2),
		adapters:      make(map[string]Adapter, 2),
		globalFilters: make([]EventFilter, 2),
		effectFilters: make(map[EventType][]TypedEventFilter, 2),
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
	// bind binary handler
	for _, adapter := range d.adapters {
		adapter.SetEventHandler(d)
	}
	return nil
}

func (d *EventDirector) OnReceived(ctx EventContext, header EventHeader, packet []byte) {
	defer func() {
		if err := recover(); err != nil {
			Log().Errorf("unexcepted panic error: %s", err)
		}
	}()
	// select filter
	effective := d.effectFilters[header.EventType]
	filters := make([]EventFilter, 0, len(d.globalFilters)+len(effective))
	copy(filters, d.globalFilters)
	for _, f := range effective {
		filters = append(filters, f)
	}
	// do filter
	_ = d.makeFilterChain(func(bh EventHeader) (err error) {
		if ctx.Async() {
			err = d.work(bh, packet)
		} else {
			err = d.workers.Process([]interface{}{bh, packet}).(error)
		}
		if err != nil {
			Log().Errorf("handle/filter event, error: %s", err)
		}
		return nil
	}, filters)(header)
}

func (d *EventDirector) Startup(ctx context.Context) error {
	return nil
}

func (d *EventDirector) Shutdown(ctx context.Context) error {
	d.workers.Close()
	return nil
}

func (d *EventDirector) work(header EventHeader, packet []byte) error {
	// format
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

func (d *EventDirector) makeFilterChain(next EventFilterInvoker, filters []EventFilter) EventFilterInvoker {
	for i := len(filters) - 1; i >= 0; i-- {
		next = filters[i].DoFilter(next)
	}
	return next
}

// InjectFormatters Auto inject
func (d *EventDirector) InjectFormatters(formatters []Transformer) {
	Log().Infof("server set formatters: %d", len(formatters))
	for _, f := range formatters {
		d.transformers[f.ActiveON()] = f
	}
}

// InjectGlobalFilters Auto inject
func (d *EventDirector) InjectGlobalFilters(filters []EventFilter) {
	Log().Infof("server add global filters: %d", len(filters))
	for _, f := range filters {
		// 忽略EffectiveFilter
		if _, is := f.(TypedEventFilter); is {
			continue
		}
		d.globalFilters = append(d.globalFilters, f)
	}
}

// InjectEffectFilters Auto inject
func (d *EventDirector) InjectEffectFilters(filters []TypedEventFilter) {
	Log().Infof("server add effect filters: %d", len(filters))
	for _, filter := range filters {
		if list, ok := d.effectFilters[filter.EffectON()]; ok {
			d.effectFilters[filter.EffectON()] = append(list, filter)
		} else {
			d.effectFilters[filter.EffectON()] = []TypedEventFilter{filter}
		}
	}
}

// InjectDispatchers Auto inject
func (d *EventDirector) InjectDispatchers(items []Dispatcher) {
	Log().Infof("server add dispatchers: %d", len(items))
	for _, a := range items {
		d.dispatchers[a.DispatcherId()] = a
	}
}

// InjectAdapters Auto inject
func (d *EventDirector) InjectAdapters(items []Adapter) {
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
