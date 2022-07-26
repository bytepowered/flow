package flow

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func BenchmarkIsNil(b *testing.B) {
	for i := 0; i < b.N; i++ {
		assert.False(b, IsNil(b))
	}
}

func BenchmarkIsNilDisabled(b *testing.B) {
	for i := 0; i < b.N; i++ {
		assert.False(b, IsNil(b))
	}
}
