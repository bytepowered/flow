package compat

import (
	"context"
	flow "github.com/bytepowered/flow/v2/pkg"
	"github.com/bytepowered/runv"
)

var _ runv.Liveness = new(Source)
var _ runv.Liveorder = new(Source)

type Source struct {
	OrderedSource
	emitters []flow.Emitter
	donefun  context.CancelFunc
	donectx  context.Context
}

func NewSource() *Source {
	dctx, dfun := context.WithCancel(context.TODO())
	return &Source{
		donectx: dctx, donefun: dfun,
	}
}

func (s *Source) Startup(ctx context.Context) error {
	runv.Assert(s.emitters != nil, "emitters is required")
	if s.donectx == nil || s.donefun == nil {
		s.donectx, s.donefun = context.WithCancel(context.TODO())
	}
	return nil
}

func (s *Source) Shutdown(ctx context.Context) error {
	s.donefun()
	return nil
}

func (s *Source) Done() <-chan struct{} {
	return s.donectx.Done()
}

func (s *Source) Context() context.Context {
	return s.donectx
}

func (s *Source) AddEmitter(e flow.Emitter) {
	s.emitters = append(s.emitters, e)
}

func (s *Source) Emit(ctx flow.StateContext, event flow.Event) {
	for _, emitter := range s.emitters {
		emitter.Emit(ctx, event)
	}
}
