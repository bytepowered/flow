package flow

import (
	"bytes"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

var _ Filter = new(ptFilter)
var _ Output = new(ptOutput)
var _ Transformer = new(ptTransformer)

type ptFilter int

func (t *ptFilter) Tag() string {
	panic("implement me")
}

func (t *ptFilter) DoFilter(next FilterFunc) FilterFunc {
	panic("implement me")
}

type ptOutput int

func (t ptOutput) Tag() string {
	panic("implement me")
}

func (t ptOutput) OnSend(ctx StateContext, events ...Event) {
	panic("implement me")
}

type ptTransformer int

func (t ptTransformer) Tag() string {
	panic("implement me")
}

func (t ptTransformer) DoTransform(ctx StateContext, in []Event) (out []Event, err error) {
	panic("implement me")
}

func TestEnsureNewPipeline(t *testing.T) {
	r := NewPipeline("in")
	r.AddFilter(new(ptFilter))
	r.AddOutput(new(ptOutput))
	r.AddTransformer(new(ptTransformer))
}

func TestUnmarshalPipelineDefinition(t *testing.T) {
	content := `
[[pipeline]]
description = "desc"
[pipeline.selector]
input = "i1"
outputs = ["o1", "o2"]
filters = ["f1", "f2"]
transformers = ["t1", "t2"]

[[pipeline]]
description = "desc"
[pipeline.selector]
input = "i1"
outputs = ["o1", "o2"]
filters = ["f1", "f2"]
transformers = ["t1", "t2"]
`
	viper.SetConfigType("toml")
	if err := viper.ReadConfig(bytes.NewReader([]byte(content))); err != nil {
		assert.Fail(t, "error: "+err.Error())
	}
	definitions := make([]Definition, 0)
	assert.Nil(t, UnmarshalConfigKey("pipeline", &definitions))
	assert.Equal(t, 2, len(definitions))
	for _, d := range definitions {
		assert.Equal(t, "desc", d.Description)
		assert.Equal(t, "i1", d.Selector.Input)
		assert.Equal(t, []string{"o1", "o2"}, d.Selector.Outputs)
		assert.Equal(t, []string{"f1", "f2"}, d.Selector.Filters)
		assert.Equal(t, []string{"t1", "t2"}, d.Selector.Transformers)
	}
}
