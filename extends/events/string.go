package events

import (
	"github.com/bytepowered/flow/v3/pkg"
	"time"
)

var _ flow.Event = new(StringEvent)

type StringEvent struct {
	Headers flow.Header
	record  string
	t       time.Time
}

func NewStringEvent(header flow.Header, data string) *StringEvent {
	return &StringEvent{
		Headers: header,
		record:  data,
		t:       time.UnixMicro(time.Duration(header.Time).Microseconds()),
	}
}

func (e *StringEvent) Tag() string {
	return e.Headers.Tag
}

func (e *StringEvent) Kind() flow.Kind {
	return e.Headers.Kind
}

func (e *StringEvent) Time() time.Time {
	return e.t
}

func (e *StringEvent) Header() flow.Header {
	return e.Headers
}

func (e *StringEvent) Record() interface{} {
	return e.record
}

func (e *StringEvent) Frames() []byte {
	return []byte(e.record)
}
