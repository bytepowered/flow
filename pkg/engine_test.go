package flow

import (
	"bytes"
	"container/list"
	"context"
	"fmt"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

var _ Filter = new(IntFilter)

type IntFilter struct {
	order int
	list  *list.List
}

func (i IntFilter) Tag() string {
	return fmt.Sprintf("%d", i.order)
}

func (i IntFilter) DoFilter(next FilterFunc) FilterFunc {
	return func(ctx StateContext, event Event) error {
		i.list.PushBack(i.order)
		return next(ctx, event)
	}
}

var _ Input = new(NopInput)

type NopInput int

func (n NopInput) Tag() string {
	return "input"
}

func (n NopInput) OnReceived(ctx context.Context, queue chan<- Event) {

}

var _ Output = new(NopOutput)

type NopOutput int

func (n NopOutput) Tag() string {
	return "output"
}

func (n NopOutput) OnSend(ctx context.Context, events ...Event) {

}

var _ Transformer = new(NopTransformer)

type NopTransformer int

func (n NopTransformer) Tag() string {
	return "transformer"
}

func (n NopTransformer) DoTransform(ctx StateContext, in []Event) (out []Event, err error) {
	return in, nil
}

var _ Filter = new(NopFilter)

type NopFilter int

func (n NopFilter) Tag() string {
	return "filter"
}

func (n NopFilter) DoFilter(next FilterFunc) FilterFunc {
	fmt.Println("call nop filter")
	return next
}

////

func TestMakeFilterChain(t *testing.T) {
	const count = 4
	orders := list.New()
	last := FilterFunc(func(ctx StateContext, event Event) error {
		orders.PushBack(count)
		return nil
	})
	filters := make([]Filter, 0, count)
	for i := 0; i < count; i++ {
		if i == 2 {
			filters = append(filters, new(NopFilter))
		}
		filters = append(filters, &IntFilter{
			order: i,
			list:  orders,
		})
	}
	err := makeFilterChain(last, filters)(nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, 1+count, orders.Len())
	for e := orders.Front(); e != nil; e = e.Next() {
		fmt.Println(e.Value)
	}
}

func TestEngineInit(t *testing.T) {
	eng := NewEventEngine(WithQueueSize(0))
	assert.Panicsf(t, func() {
		_ = eng.OnInit()
	}, "must panic")
	eng.AddInput(new(NopInput))
	eng.AddOutput(new(NopOutput))
	assert.Nil(t, eng.OnInit())
	assert.Nil(t, eng.Startup(context.TODO()))
	assert.Nil(t, eng.Shutdown(context.TODO()))
}

func TestEngineWorkModePipeline(t *testing.T) {
	eng := NewEventEngine()
	content := `
[[pipeline]]
description = "desc"
[pipeline.selector]
input = "input"
outputs = ["output"]
filters = ["filter"]
transformers = ["transformer"]
`
	viper.SetConfigType("toml")
	if err := viper.ReadConfig(bytes.NewReader([]byte(content))); err != nil {
		assert.Fail(t, "error: "+err.Error())
	}
	in := new(NopInput)
	out := new(NopOutput)
	f := new(NopFilter)
	tf := new(NopTransformer)
	eng.AddInput(in)
	eng.AddOutput(out)
	eng.AddFilter(f)
	eng.AddTransformer(tf)
	assert.Nil(t, eng.OnInit())
	ps := eng.GetPipelines()
	assert.Equal(t, 1, len(ps))
	assert.Equal(t, in.Tag(), ps[0].Input)
	assert.Equal(t, out, ps[0].outputs[0])
	assert.Equal(t, f, ps[0].filters[0])
	assert.Equal(t, tf, ps[0].transformers[0])
}

func TestEngineWorkModeSingle(t *testing.T) {
	eng := NewEventEngine()
	content := `
[engine]
workmode = "single"
`
	viper.SetConfigType("toml")
	if err := viper.ReadConfig(bytes.NewReader([]byte(content))); err != nil {
		assert.Fail(t, "error: "+err.Error())
	}
	in := new(NopInput)
	out := new(NopOutput)
	f := new(NopFilter)
	tf := new(NopTransformer)
	eng.AddInput(in)
	eng.AddOutput(out)
	eng.AddFilter(f)
	eng.AddTransformer(tf)
	assert.Nil(t, eng.OnInit())
	ps := eng.GetPipelines()
	assert.Equal(t, 1, len(ps))
	assert.Equal(t, in.Tag(), ps[0].Input)
	assert.Equal(t, out, ps[0].outputs[0])
	assert.Equal(t, f, ps[0].filters[0])
	assert.Equal(t, tf, ps[0].transformers[0])
}

func TestEngineExecute(t *testing.T) {
	content := `
[engine]
workmode = "single"
[engine.logger]
caller = false
format = "text"
level = "warn"
`
	viper.SetConfigType("toml")
	assert.Nil(t, SetupWithConfig(bytes.NewReader([]byte(content))), "setup")
	Register(new(NopInput))
	Register(new(NopOutput))
	Register(new(NopFilter))
	Register(new(NopTransformer))
	Execute()
}

var _ Input = new(CountInput)

type CountInput struct {
	count int
}

func (w CountInput) Tag() string {
	return "number"
}

func (w CountInput) OnReceived(ctx context.Context, queue chan<- Event) {
	for i := 0; i < w.count; i++ {
		queue <- NewStringEvent(
			Header{}, fmt.Sprintf("no: %d", i))
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
	in := &CountInput{count: 100}
	out := &CountOutput{}
	Register(out)
	Register(in)
	Execute(WithQueueSize(10))
	assert.Equal(t, in.count, out.count)
}
