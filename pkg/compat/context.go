package compat

import (
	"context"
	"fmt"
	"github.com/bytepowered/flow/v2/pkg"
	"time"
)

var _ flow.StateContext = new(StatedContext)

var statekey = struct {
	v string
}{v: fmt.Sprintf("@hide.state.key:%d", time.Now().Nanosecond())}

type StatedContext struct {
	ctx context.Context
}

func NewStatedContext(ctx context.Context, state flow.State) flow.StateContext {
	return &StatedContext{
		ctx: context.WithValue(ctx, statekey, state),
	}
}

func (e *StatedContext) Var(key interface{}) interface{} {
	return e.ctx.Value(key)
}

func (e *StatedContext) VarE(key interface{}) (interface{}, bool) {
	v := e.ctx.Value(key)
	return v, v != nil
}

func (e *StatedContext) Context() context.Context {
	return e.ctx
}

func (e *StatedContext) State() flow.State {
	return e.Var(statekey).(flow.State)
}
