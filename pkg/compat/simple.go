package compat

import flow "github.com/bytepowered/flow/v2/pkg"

type SimpleSource struct {
	emitter flow.Emitter
}

func NewSimpleSource() *SimpleSource {
	return &SimpleSource{}
}

func (a *SimpleSource) SetEmitter(aeh flow.Emitter) {
	a.emitter = aeh
}

func (a *SimpleSource) Emit(ctx flow.StateContext, record flow.Event) {
	a.emitter.Emit(ctx, record)
}
