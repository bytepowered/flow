package events

import (
	flow "github.com/bytepowered/flow/v3/pkg"
)

var _ flow.Event = new(StringFieldsEvent)

type StringFieldsEvent struct {
	*flow.ObjectEvent
}

func NewStringFieldsEvent(header flow.Header, fields []string) *StringFieldsEvent {
	return &StringFieldsEvent{
		ObjectEvent: flow.NewObjectEvent(header, fields),
	}
}

func (l *StringFieldsEvent) Fields() []string {
	return l.Record().([]string)
}
