package input

import (
	"context"
	flow "github.com/bytepowered/flow/v3/pkg"
	"github.com/bytepowered/runv"
)

var _ runv.Liveness = new(BaseInput)
var _ runv.Liveorder = new(BaseInput)
var _ flow.Input = new(BaseInput)

type BaseInput struct {
	flow.OrderedInput
	downfun context.CancelFunc
	downctx context.Context
	tag     string
}

func NewBaseInput(tag string) *BaseInput {
	dctx, dfun := context.WithCancel(context.TODO())
	return &BaseInput{
		downctx: dctx, downfun: dfun,
		tag: tag,
	}
}

func (i *BaseInput) Startup(ctx context.Context) error {
	if i.downctx == nil || i.downfun == nil {
		i.downctx, i.downfun = context.WithCancel(ctx)
	}
	return nil
}

func (i *BaseInput) Shutdown(ctx context.Context) error {
	i.downfun()
	return nil
}

func (i *BaseInput) Done() <-chan struct{} {
	return i.downctx.Done()
}

func (i *BaseInput) Context() context.Context {
	return i.downctx
}

func (i *BaseInput) Tag() string {
	return i.tag
}

func (i *BaseInput) SetTag(tag string) {
	i.tag = tag
}

func (i *BaseInput) OnReceived(ctx context.Context, queue chan<- flow.Event) {
	panic("not yet implemented")
}
