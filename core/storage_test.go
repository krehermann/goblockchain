package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemStore_Get(t *testing.T) {
	ms := NewGenericMemStore[*Header, *Block]()

	nPuts := 5
	want := make([]*Block, nPuts)
	for i := 0; i < nPuts; i++ {
		b := randomBlockWithoutPreviousBlock(t, uint32(i))
		want[i] = b
		assert.NoError(t, ms.Put(b.Header, b))

		for j := 0; j <= i; j++ {
			gotBlock, err := ms.Get(want[j].Header)
			assert.NoError(t, err)
			assert.Equal(t, want[j], gotBlock)

		}
	}
}
