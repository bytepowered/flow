package compat

import flow "github.com/bytepowered/flow/v2/pkg"

var _ flow.SourceAdapter = new(SourceAdapter)

type SourceAdapter struct {
	typeid  string
	tag     string
	emitter flow.EventEmitter
}

func NewSourceAdapter(typeid, tag string) *SourceAdapter {
	return &SourceAdapter{typeid: typeid, tag: tag}
}

func (a *SourceAdapter) TypeId() string {
	return a.typeid
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
