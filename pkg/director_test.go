package flow

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var _ EventFilter = new(tfilter)

type tfilter struct {
}

func (t *tfilter) DoFilter(next EventFilterInvoker) EventFilterInvoker {
	panic("implement me")
}

func TestFilterCopy(t *testing.T) {
	src := make([]EventFilter, 0, 4)
	src = append(src, new(tfilter))
	dist := make([]EventFilter, len(src), len(src))
	copy(dist, src)
	assert.Equal(t, len(src), len(dist))
}
