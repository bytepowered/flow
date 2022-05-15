package flow

import (
	"bytes"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

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
	ps := eng.Pipelines()
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
	ps := eng.Pipelines()
	assert.Equal(t, 1, len(ps))
	assert.Equal(t, in.Tag(), ps[0].Input)
	assert.Equal(t, out, ps[0].outputs[0])
	assert.Equal(t, f, ps[0].filters[0])
	assert.Equal(t, tf, ps[0].transformers[0])
}
