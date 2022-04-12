package input

import (
	"context"
	flow "github.com/bytepowered/flow/v3/pkg"
	"github.com/bytepowered/runv"
)

var _ runv.Liveness = new(StatedInput)
var _ runv.Liveorder = new(StatedInput)
var _ flow.Input = new(StatedInput)

type StatedInput struct {
	flow.OrderedInput
	downfun context.CancelFunc
	downctx context.Context
	tag     string
}

func NewStatedInput(tag string) *StatedInput {
	dctx, dfun := context.WithCancel(context.TODO())
	return &StatedInput{
		downctx: dctx, downfun: dfun,
		tag: tag,
	}
}

func (i *StatedInput) Startup(ctx context.Context) error {
	if i.downctx == nil || i.downfun == nil {
		i.downctx, i.downfun = context.WithCancel(ctx)
	}
	return nil
}

func (i *StatedInput) Shutdown(ctx context.Context) error {
	i.downfun()
	return nil
}

func (i *StatedInput) Done() <-chan struct{} {
	return i.downctx.Done()
}

func (i *StatedInput) Context() context.Context {
	return i.downctx
}

func (i *StatedInput) Tag() string {
	return i.tag
}

func (i *StatedInput) SetTag(tag string) {
	i.tag = tag
}

func (i *StatedInput) OnReceived(ctx context.Context, queue chan<- flow.Event) {
	panic("StatedInput(ABSTRACT): not yet implemented")
}
