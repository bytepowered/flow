package flow

import (
	"container/list"
	"context"
	"fmt"
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
	return "nop"
}

func (n NopInput) OnReceived(ctx context.Context, queue chan<- Event) {

}

var _ Output = new(NopOutput)

type NopOutput int

func (n NopOutput) Tag() string {
	return "nop"
}

func (n NopOutput) OnSend(ctx context.Context, events ...Event) {

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
	eng := NewEventEngine(WithQueueSize(1))
	assert.Panicsf(t, func() {
		_ = eng.OnInit()
	}, "must panic")
	eng.AddInput(new(NopInput))
	eng.AddOutput(new(NopOutput))
	assert.Nil(t, eng.OnInit())
	assert.Nil(t, eng.Startup(context.TODO()))
	assert.Nil(t, eng.Shutdown(context.TODO()))
}
