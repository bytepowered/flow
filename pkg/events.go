package flow

import (
	"bytes"
	"time"
)

var _ Event = NewObjectEvent(Header{}, nil)

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

//// Bytes

var _ Event = NewBytesEvent(Header{}, nil)

type BytesEvent struct {
	*ObjectEvent
}

func NewBytesEvent(header Header, data []byte) *BytesEvent {
	return &BytesEvent{
		ObjectEvent: NewObjectEvent(header, data),
	}
}

func (e *BytesEvent) Bytes() []byte {
	return e.Record().([]byte)
}

//// Text

var _ Event = NewTextEvent(Header{}, "")

type TextEvent struct {
	*ObjectEvent
}

func NewTextEvent(header Header, data string) *TextEvent {
	return &TextEvent{
		ObjectEvent: NewObjectEvent(header, data),
	}
}

func (e *TextEvent) Text() string {
	return e.Record().(string)
}

//// Fields

var _ Event = NewFieldsEvent(Header{}, nil)

type FieldsEvent struct {
	*ObjectEvent
}

func NewFieldsEvent(header Header, data []string) *FieldsEvent {
	return &FieldsEvent{
		ObjectEvent: NewObjectEvent(header, data),
	}
}

func (e *FieldsEvent) Fields() []string {
	return e.Record().([]string)
}

//// Buffer

var _ Event = NewBufferEvent(Header{}, nil)

type BufferEvent struct {
	*ObjectEvent
}

func NewBufferEvent(header Header, data *bytes.Buffer) *BufferEvent {
	return &BufferEvent{
		ObjectEvent: NewObjectEvent(header, data),
	}
}

func (e *BufferEvent) Buffer() *bytes.Buffer {
	return e.Record().(*bytes.Buffer)
}
