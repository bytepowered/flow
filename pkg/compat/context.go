package compat

import (
	"context"
	"github.com/bytepowered/flow/pkg"
)

var _ flow.EventContext = new(EventContext)

var statekey = struct {
	v string
}{"state.key"}

type EventContext struct {
	ctx context.Context
}

func NewEventContext(ctx context.Context, state flow.EventState) flow.EventContext {
	return &EventContext{
		ctx: context.WithValue(ctx, statekey, state),
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

func (e *EventContext) State() flow.EventState {
	return e.Var(statekey).(flow.EventState)
}
