package flow

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKindNameMapper(t *testing.T) {
	assert.NotNil(t, KindNameMapper)
	for i := 0; i < 100; i++ {
		n := fmt.Sprintf("ERR%3d", i)
		KindNameMapper.SetName(Kind(i), n)
		assert.Equal(t, n, KindNameMapper.GetName(Kind(i)))
	}
	assert.Equal(t, "undefined", KindNameMapper.GetName(Kind(9999)))
}
