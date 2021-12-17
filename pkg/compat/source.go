package compat

import (
	"context"
	flow "github.com/bytepowered/flow/v2/pkg"
	"github.com/bytepowered/runv"
)

var _ runv.Liveness = new(SlimSource)
var _ runv.Liveorder = new(SlimSource)

type SlimSource struct {
	emitters []flow.Emitter
	donefun  context.CancelFunc
	donectx  context.Context
}

func NewSlimSource() *SlimSource {
	dctx, dfun := context.WithCancel(context.TODO())
	return &SlimSource{
		donectx: dctx, donefun: dfun,
	}
}

func (s *SlimSource) Startup(ctx context.Context) error {
	runv.Assert(s.emitters != nil, "emitters is required")
	if s.donectx == nil || s.donefun == nil {
		s.donectx, s.donefun = context.WithCancel(context.TODO())
	}
	return nil
}

func (s *SlimSource) Shutdown(ctx context.Context) error {
	s.donefun()
	return nil
}

func (s *SlimSource) Done() <-chan struct{} {
	return s.donectx.Done()
}

func (s *SlimSource) Context() context.Context {
	return s.donectx
}

func (s *SlimSource) AddEmitter(e flow.Emitter) {
	s.emitters = append(s.emitters, e)
}

func (s *SlimSource) Emit(ctx flow.StateContext, event flow.Event) {
	for _, emitter := range s.emitters {
		emitter.Emit(ctx, event)
	}
}

func (*SlimSource) Order(state runv.State) int {
	// Source: 关闭优先，启动靠后
	switch state {
	case runv.StateShutdown:
		return -1000
	case runv.StateStartup:
		return 1000
	default:
		return 0
	}
}
