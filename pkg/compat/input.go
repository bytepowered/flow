package compat

import (
	"context"
	flow "github.com/bytepowered/flow/v2/pkg"
	"github.com/bytepowered/runv"
)

var _ runv.Liveness = new(Input)
var _ runv.Liveorder = new(Input)
var _ flow.Input = new(Input)

type Input struct {
	OrderedInput
	emitters []flow.Emitter
	downfun  context.CancelFunc
	downctx  context.Context
	tag      string
}

func NewInput(tag string) *Input {
	dctx, dfun := context.WithCancel(context.TODO())
	return &Input{
		downctx: dctx, downfun: dfun,
		tag: tag,
	}
}

func (i *Input) Startup(ctx context.Context) error {
	runv.Assert(i.emitters != nil, "emitters is required")
	if i.downctx == nil || i.downfun == nil {
		i.downctx, i.downfun = context.WithCancel(ctx)
	}
	return nil
}

func (i *Input) Shutdown(ctx context.Context) error {
	i.downfun()
	return nil
}

func (i *Input) Done() <-chan struct{} {
	return i.downctx.Done()
}

func (i *Input) Context() context.Context {
	return i.downctx
}

func (i *Input) AddEmitter(e flow.Emitter) {
	i.emitters = append(i.emitters, e)
}

func (i *Input) Tag() string {
	return i.tag
}

func (i *Input) SetTag(tag string) {
	i.tag = tag
}

func (i *Input) Emit(ctx flow.StateContext, event flow.Event) {
	for _, emitter := range i.emitters {
		emitter.Emit(ctx, event)
	}
}
