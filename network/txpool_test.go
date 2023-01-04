package network

import (
	"testing"

	"github.com/krehermann/goblockchain/core"
	"github.com/stretchr/testify/assert"
)

func TestTxPool(t *testing.T) {
	p := NewTxPool()
	assert.Equal(t, 0, p.Len())

	tx := core.NewTransaction([]byte("test data"))
	ok, err := p.Add(tx, &core.DefaultTxHasher{})
	assert.True(t, ok)
	assert.NoError(t, err)
	assert.Equal(t, 1, p.Len())

	assert.True(t, p.Has(tx.Hash(&core.DefaultTxHasher{})))

	// no dupes
	ok, err = p.Add(tx, &core.DefaultTxHasher{})
	assert.False(t, ok)
	assert.NoError(t, err)
	assert.Equal(t, 1, p.Len())

	p.Flush()
	assert.Equal(t, 0, p.Len())

}
