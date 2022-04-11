package flow

import "time"

var _ Event = new(ObjectEvent)

type ObjectEvent struct {
	Headers Header
	object  interface{}
	time    time.Time
}

func NewObjectEvent(header Header, object interface{}) *ObjectEvent {
	return &ObjectEvent{
		Headers: header,
		object:  object,
		time:    time.UnixMicro(time.Duration(header.Time).Microseconds()),
	}
}

func (e *ObjectEvent) ID() int64 {
	return e.Headers.Id
}

func (e *ObjectEvent) Tag() string {
	return e.Headers.Tag
}

func (e *ObjectEvent) Kind() Kind {
	return e.Headers.Kind
}

func (e *ObjectEvent) Time() time.Time {
	return e.time
}

func (e *ObjectEvent) Header() Header {
	return e.Headers
}

func (e *ObjectEvent) Record() interface{} {
	return e.object
}

//// String

var _ Event = new(StringEvent)

type StringEvent struct {
	*ObjectEvent
}

func NewStringEvent(header Header, data string) *StringEvent {
	return &StringEvent{
		ObjectEvent: NewObjectEvent(header, data),
	}
}

func (e *StringEvent) String() string {
	return e.Record().(string)
}
