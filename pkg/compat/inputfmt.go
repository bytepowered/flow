package compat

import (
	"context"
	flow "github.com/bytepowered/flow/v2/pkg"
	"github.com/bytepowered/runv"
)

type FormatInput struct {
	*Input
	formatter flow.Formatter
	tag       string
}

func NewFormatInput(tag string, f flow.Formatter) *FormatInput {
	return &FormatInput{
		Input:     NewInput(tag),
		formatter: f,
		tag:       tag,
	}
}

func (i *FormatInput) SetFormatter(f flow.Formatter) {
	i.formatter = f
}

func (i *FormatInput) EnsureDeps() {
	i.tag = i.Input.Tag()
	runv.Assert(i.formatter != nil, "formatter is required")
}

func (i *FormatInput) Startup(ctx context.Context) error {
	i.EnsureDeps()
	return i.Input.Startup(ctx)
}

func (i *FormatInput) EmitFrame(ctx flow.StateContext, data []byte) error {
	evt, err := i.formatter.DoFormat(ctx.Context(), i.Input.Tag(), data)
	if err != nil {
		return err
	}
	// skip nil events
	if evt != nil || runv.IsNil(evt) {
		i.Emit(ctx, evt)
	}
	return nil
}
