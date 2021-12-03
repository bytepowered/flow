package compat

import (
	"context"
	"github.com/bytepowered/flow/pkg"
)

var _ flow.EventContext = new(EventContext)

type EventContext struct {
	ctx   context.Context
	async bool
}

func NewEventContext(ctx context.Context, async bool) flow.EventContext {
	return &EventContext{
		ctx:   ctx,
		async: async,
	}
}

func (e *EventContext) Var(key interface{}) interface{} {
	return e.ctx.Value(key)
}

func (e *EventContext) VarE(key interface{}) (interface{}, bool) {
	v := e.ctx.Value(key)
	return v, v != nil
}

func (e *EventContext) Context() context.Context {
	return e.ctx
}

func (e *EventContext) Async() bool {
	return e.async
}
