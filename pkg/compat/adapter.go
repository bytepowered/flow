package compat

import flow "github.com/bytepowered/flow/v2/pkg"

type AbcSourceAdapter struct {
	emitter flow.EventEmitter
}

func NewAbcSourceAdapter() *AbcSourceAdapter {
	return &AbcSourceAdapter{}
}

func (a *AbcSourceAdapter) SetEmitter(aeh flow.EventEmitter) {
	a.emitter = aeh
}

func (a *AbcSourceAdapter) Emit(ctx flow.EventContext, record flow.EventRecord) {
	a.emitter.Emit(ctx, record)
}
