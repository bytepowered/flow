package flow

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

var _ Filter = new(matchFilter)
var _ Output = new(matchOutput)
var _ Transformer = new(matchTransformer)

type matchFilter int

func (t *matchFilter) Tag() string {
	return "net.filter"
}

func (t *matchFilter) DoFilter(next FilterFunc) FilterFunc {
	panic("implement me")
}

type matchOutput int

func (t matchOutput) Tag() string {
	return "std.output"
}

func (t matchOutput) OnSend(ctx context.Context, events ...Event) {
	panic("implement me")
}

type matchTransformer int

func (t matchTransformer) Tag() string {
	return "transformer"
}

func (t matchTransformer) DoTransform(ctx StateContext, in []Event) (out []Event, err error) {
	panic("implement me")
}

func TestMatchMatchFilter(t *testing.T) {
	doTestMatcher(t, []string{"net.filter"}, 1)
}

func TestMatchMatchOutput(t *testing.T) {
	doTestMatcher(t, []string{"std.output"}, 1)
}

func TestMatchNotFound(t *testing.T) {
	doTestMatcher(t, []string{"0"}, 0)
	doTestMatcher(t, []string{"0", "net.filter"}, 1)
	doTestMatcher(t, []string{"ab", "std.output.xx"}, 0)
}

func TestMatchPrefixWildcard(t *testing.T) {
	doTestMatcher(t, []string{"net.*", "std.output"}, 2)
}

func TestMatchWildcard(t *testing.T) {
	doTestMatcher(t, []string{"*"}, 3)
}

func doTestMatcher(t *testing.T, tags []string, expected int) {
	wc := newMatcher(tags)
	count := 0
	wc.on([]interface{}{
		new(matchFilter), new(matchOutput), new(matchTransformer),
	}, func(tag string, plugin interface{}) {
		assert.NotNil(t, plugin, "accepted plugin must not nil")
		count++
	})
	assert.Equal(t, expected, count, "count not match")
}
