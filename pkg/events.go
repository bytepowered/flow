package flow

import (
	"bytes"
	"time"
)

var _ Event = new(BufferedEvent)

type BufferedEvent struct {
	Headers Header
	buffer  *bytes.Buffer
	t       time.Time
}

func BufferedEventOf(header Header, data []byte) *BufferedEvent {
	return NewBufferedEvent(header, bytes.NewBuffer(data))
}

func NewBufferedEvent(header Header, buffer *bytes.Buffer) *BufferedEvent {
	return &BufferedEvent{
		Headers: header,
		buffer:  buffer,
		t:       time.UnixMicro(time.Duration(header.Time).Microseconds()),
	}
}

func (e BufferedEvent) Tag() string {
	return e.Headers.Tag
}

func (e BufferedEvent) Kind() Kind {
	return e.Headers.Kind
}

func (e BufferedEvent) Time() time.Time {
	return e.t
}

func (e BufferedEvent) Header() Header {
	return e.Headers
}

func (e BufferedEvent) Record() interface{} {
	return e.buffer
}

func (e BufferedEvent) Frames() []byte {
	return e.buffer.Bytes()
}

func (e BufferedEvent) Buffer() *bytes.Buffer {
	return e.buffer
}
