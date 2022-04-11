package events

import (
	"bytes"
	"github.com/bytepowered/flow/v3/pkg"
)

var _ flow.Event = new(BufferedEvent)

type BufferedEvent struct {
	*flow.ObjectEvent
}

func BufferedEventOf(header flow.Header, data []byte) *BufferedEvent {
	return NewBufferedEvent(header, bytes.NewBuffer(data))
}

func NewBufferedEvent(header flow.Header, buffer *bytes.Buffer) *BufferedEvent {
	return &BufferedEvent{
		ObjectEvent: flow.NewObjectEvent(header, buffer),
	}
}

func (e BufferedEvent) Bytes() []byte {
	return e.Buffer().Bytes()
}

func (e BufferedEvent) Buffer() *bytes.Buffer {
	return e.Record().(*bytes.Buffer)
}
