package events

import (
	"bytes"
	"github.com/bytepowered/flow/v3/pkg"
	"time"
)

var _ flow.Event = new(BufferedEvent)

type BufferedEvent struct {
	Headers flow.Header
	buffer  *bytes.Buffer
	t       time.Time
}

func BufferedEventOf(header flow.Header, data []byte) *BufferedEvent {
	return NewBufferedEvent(header, bytes.NewBuffer(data))
}

func NewBufferedEvent(header flow.Header, buffer *bytes.Buffer) *BufferedEvent {
	return &BufferedEvent{
		Headers: header,
		buffer:  buffer,
		t:       time.UnixMicro(time.Duration(header.Time).Microseconds()),
	}
}

func (e BufferedEvent) Tag() string {
	return e.Headers.Tag
}

func (e BufferedEvent) Kind() flow.Kind {
	return e.Headers.Kind
}

func (e BufferedEvent) Time() time.Time {
	return e.t
}

func (e BufferedEvent) Header() flow.Header {
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
