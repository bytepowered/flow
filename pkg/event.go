package flow

import "context"

var _ EventContext = new(econtext)

type econtext struct {
	ctx   context.Context
	async bool
}

func NewAsyncEventContext(ctx context.Context) EventContext {
	return NewEventContext(ctx, true)
}

func NewSyncEventContext(ctx context.Context) EventContext {
	return NewEventContext(ctx, false)
}

func (e *econtext) GetVar(key interface{}) interface{} {
	return e.ctx.Value(key)
}

func (e *econtext) GetVarE(key interface{}) (interface{}, bool) {
	v := e.ctx.Value(key)
	return v, v != nil
}

func NewEventContext(ctx context.Context, async bool) EventContext {
	return &econtext{
		ctx:   ctx,
		async: async,
	}
}

func (e *econtext) Context() context.Context {
	return e.ctx
}

func (e *econtext) Async() bool {
	return e.async
}
