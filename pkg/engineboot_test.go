package flow

import (
	"bytes"
	"context"
	"fmt"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

var _ Input = new(CountInput)

type CountInput struct {
	count int
}

func (w CountInput) Tag() string {
	return "number"
}

func (w CountInput) OnRead(ctx context.Context, queue chan<- Event) {
	for i := 0; i < w.count; i++ {
		queue <- NewTextEvent(
			Header{Tag: w.Tag()}, fmt.Sprintf("no: %d", i))
	}
}

var _ Output = new(CountOutput)

type CountOutput struct {
	count int
}

func (o *CountOutput) Tag() string {
	return "wait"
}

func (o *CountOutput) OnSend(ctx context.Context, events ...Event) {
	for range events {
		o.count++
	}
}

func (o *CountOutput) Reset() {
	o.count = 0
}

func TestEngineServe(t *testing.T) {
	content := `
[engine]
workmode = "single"
[engine.logger]
caller = false
format = "text"
level = "error"
`
	viper.SetConfigType("toml")
	assert.Nil(t, SetupWithConfig(bytes.NewReader([]byte(content))), "setup")
	in := &CountInput{count: 20}
	out := &CountOutput{}
	Register(out)
	Register(in)
	Execute(WithQueueSize(10))
	assert.Equal(t, in.count, out.count)
}
