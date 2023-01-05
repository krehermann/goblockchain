package network

import (
	"fmt"
	"testing"
	"time"

	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/types"
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

func TestTxPool_Add(t *testing.T) {
	p := NewTxPool()

	nTx := 12
	for i := 0; i < nTx; i += 1 {
		d := fmt.Sprintf("tx-%d", i)
		tx := core.NewTransaction([]byte(d))
		tx.SetCreatedAt(time.Now())
		p.Add(tx, &core.DefaultTxHasher{})
	}
	assert.Equal(t, p.Len(), nTx, "pool len %d != %d", p.Len(), nTx)

	txns := p.Transactions()
	for i := 0; i < p.Len()-1; i += 1 {
		assert.True(t, txns[i].GetCreatedAt().Before(txns[i+1].GetCreatedAt()))
	}
}

func TestTxMapSorter(t *testing.T) {
	txn1 := core.NewTransaction([]byte("first"))
	txn1 = txn1.SetCreatedAt(time.Unix(0, 0))
	txn2 := core.NewTransaction([]byte("second"))
	txn2 = txn2.SetCreatedAt(time.Now())

	m := map[types.Hash]*core.Transaction{
		txn2.Hash(&core.DefaultTxHasher{}): txn2,
		txn1.Hash(&core.DefaultTxHasher{}): txn1,
	}

	s := NewTxMapSorter(m)
	sorted := s.Get()
	assert.Len(t, sorted, 2)
	assert.Equal(t, txn1, sorted[0])
	assert.Equal(t, txn2, sorted[1])

}
