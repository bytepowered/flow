package flow

import (
	"bytes"
	"context"
	"fmt"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

var _ Input = new(CounterInput)

type CounterInput struct {
	count int
}

func (w CounterInput) Tag() string {
	return "counter"
}

func (w CounterInput) OnRead(ctx context.Context, queue chan<- Event) {
	for i := 0; i < w.count; i++ {
		queue <- NewTextEvent(
			Header{Tag: w.Tag()}, fmt.Sprintf("no: %d", i))
	}
}

var _ Output = new(CounterOutput)

type CounterOutput struct {
	count int
}

func (o *CounterOutput) Tag() string {
	return "counter"
}

func (o *CounterOutput) OnSend(ctx StateContext, events ...Event) {
	for range events {
		o.count++
	}
}

func (o *CounterOutput) Reset() {
	o.count = 0
}

func TestSingleEngineServe(t *testing.T) {
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
	in := &CounterInput{count: 20}
	out := &CounterOutput{}
	Register(out)
	Register(in)
	Execute(WithQueueSize(10))
	assert.Equal(t, in.count, out.count)
}
