package flow

import (
	"bytes"
	"context"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
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
	r := NewRouter("in")
	r.AddFilter(new(routeFilter))
	r.AddOutput(new(routeOutput))
	r.AddTransformer(new(routeTransformer))
}

func TestUnmarshalRouterGroupDefinition(t *testing.T) {
	content := `
[[router]]
description = "desc"
[router.selector]
input = "i1"
outputs = ["o1", "o2"]
filters = ["f1", "f2"]
transformers = ["t1", "t2"]

[[router]]
description = "desc"
[router.selector]
input = "i1"
outputs = ["o1", "o2"]
filters = ["f1", "f2"]
transformers = ["t1", "t2"]
`
	viper.SetConfigType("toml")
	if err := viper.ReadConfig(bytes.NewReader([]byte(content))); err != nil {
		assert.Fail(t, "error: "+err.Error())
	}
	definitions := make([]RouteGroupDefinition, 0)
	assert.Nil(t, UnmarshalConfigKey("router", &definitions))
	assert.Equal(t, 2, len(definitions))
	for _, d := range definitions {
		assert.Equal(t, "desc", d.Description)
		assert.Equal(t, "i1", d.Selector.Input)
		assert.Equal(t, []string{"o1", "o2"}, d.Selector.OutputTags)
		assert.Equal(t, []string{"f1", "f2"}, d.Selector.FilterTags)
		assert.Equal(t, []string{"t1", "t2"}, d.Selector.TransformerTags)
	}
}
