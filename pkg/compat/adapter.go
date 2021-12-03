package compat

import flow "github.com/bytepowered/flow/pkg"

var _ flow.SourceAdapter = new(SourceAdapter)

type SourceAdapter struct {
	tag     string
	emitter flow.EventEmitter
}

func NewSourceAdapter(tag string) *SourceAdapter {
	return &SourceAdapter{tag: tag}
}

func (a *SourceAdapter) Tag() string {
	return a.tag
}

func (a *SourceAdapter) SetEmitter(aeh flow.EventEmitter) {
	a.emitter = aeh
}

func (a *SourceAdapter) Emit(ctx flow.EventContext, record flow.EventRecord) {
	a.emitter.Emit(ctx, record)
}
