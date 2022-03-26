package flow

import (
	"context"
	"testing"
)

var _ Filter = new(routeFilter)
var _ Output = new(routeOutput)
var _ Transformer = new(routeTransformer)

type routeFilter int

func (t *routeFilter) Tag() string {
	panic("implement me")
}

func (t *routeFilter) DoFilter(next FilterFunc) FilterFunc {
	panic("implement me")
}

type routeOutput int

func (t routeOutput) Tag() string {
	panic("implement me")
}

func (t routeOutput) OnSend(ctx context.Context, events ...Event) {
	panic("implement me")
}

type routeTransformer int

func (t routeTransformer) Tag() string {
	panic("implement me")
}

func (t routeTransformer) DoTransform(ctx StateContext, in []Event) (out []Event, err error) {
	panic("implement me")
}

func TestEnsureNewRouter(t *testing.T) {
	r := NewRouter()
	r.AddFilter(new(routeFilter))
	r.AddOutput(new(routeOutput))
	r.AddTransformer(new(routeTransformer))
}
