package events

import (
	flow "github.com/bytepowered/flow/v3/pkg"
	"time"
)

var _ flow.Event = new(StringFieldsEvent)

type StringFieldsEvent struct {
	header flow.Header
	time   time.Time
	fields []string
}

func NewStringFieldsEvent(event flow.Event, fields []string) *StringFieldsEvent {
	return &StringFieldsEvent{
		header: event.Header(), time: event.Time(),
		fields: fields,
	}
}

func (l *StringFieldsEvent) Tag() string {
	return l.header.Tag
}

func (l *StringFieldsEvent) Kind() flow.Kind {
	return l.header.Kind
}

func (l *StringFieldsEvent) Time() time.Time {
	return l.time
}

func (l *StringFieldsEvent) Header() flow.Header {
	return l.header
}

func (l *StringFieldsEvent) Record() interface{} {
	return l.fields
}

func (l *StringFieldsEvent) Fields() []string {
	return l.fields
}

func (l *StringFieldsEvent) Frames() []byte {
	panic("StringFieldsEvent: Frames() is not yet implemented")
}
