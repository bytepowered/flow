package compat

import (
	"context"
	flow "github.com/bytepowered/flow/v2/pkg"
	"github.com/bytepowered/runv"
)

var _ runv.Liveness = new(SlimSource)

type SlimSource struct {
	emitter flow.Emitter
	donef   context.CancelFunc
	donec   context.Context
}

func NewSlimSource() *SlimSource {
	dctx, dfun := context.WithCancel(context.TODO())
	return &SlimSource{
		donec: dctx, donef: dfun,
	}
}

func (s *SlimSource) Startup(ctx context.Context) error {
	runv.Assert(s.emitter != nil, "emitter is required")
	if s.donec == nil || s.donef == nil {
		s.donec, s.donef = context.WithCancel(context.TODO())
	}
	return nil
}

func (s *SlimSource) Shutdown(ctx context.Context) error {
	s.donef()
	return nil
}

func (s *SlimSource) Done() <-chan struct{} {
	return s.donec.Done()
}

func (s *SlimSource) Context() context.Context {
	return s.donec
}

func (s *SlimSource) SetEmitter(e flow.Emitter) {
	s.emitter = e
}

func (s *SlimSource) Emit(ctx flow.StateContext, event flow.Event) {
	s.emitter.Emit(ctx, event)
}
