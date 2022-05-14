package flow

import (
	"context"
	"fmt"
	"time"
)

var _ StateContext = new(StatefulContext)

var hiddenKey = struct {
	v string
}{v: fmt.Sprintf("@internal.state.key:%d", time.Now().Nanosecond())}

type StatefulContext struct {
	ctx context.Context
}

func NewStateContext(ctx context.Context, state State) StateContext {
	return &StatefulContext{
		ctx: context.WithValue(ctx, hiddenKey, state),
	}
}

func (e *StatefulContext) Var(key interface{}) interface{} {
	return e.ctx.Value(key)
}

func (e *StatefulContext) VarE(key interface{}) (interface{}, bool) {
	v := e.ctx.Value(key)
	return v, v != nil
}

func (e *StatefulContext) Context() context.Context {
	return e.ctx
}

func (e *StatefulContext) State() State {
	return e.Var(hiddenKey).(State)
}
