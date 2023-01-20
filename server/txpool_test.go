package server

import (
	"fmt"
	"testing"
	"time"

	"github.com/krehermann/goblockchain/core"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestTxPool(t *testing.T) {
	p := NewTxPool()
	assert.Equal(t, 0, p.PendingCount())

	tx := core.NewTransaction([]byte("test data"))
	ok, err := p.Add(tx, &core.DefaultTxHasher{})
	assert.True(t, ok)
	assert.NoError(t, err)
	assert.Equal(t, 1, p.PendingCount())

	assert.True(t, p.Contains(tx.Hash(&core.DefaultTxHasher{})))

	// no dupes
	ok, err = p.Add(tx, &core.DefaultTxHasher{})
	assert.False(t, ok)
	assert.NoError(t, err)
	assert.Equal(t, 1, p.PendingCount())

	p.ClearPending()
	assert.Equal(t, 0, p.PendingCount())

}

func TestTxPool_Add(t *testing.T) {
	nTx := 12

	l, err := zap.NewDevelopment()
	assert.NoError(t, err)
	p := NewTxPool(WithLogger(l), MaxPoolDepth(nTx))

	for i := 0; i < nTx; i += 1 {
		d := fmt.Sprintf("tx-%d", i)
		tx := core.NewTransaction([]byte(d))
		tx.SetCreatedAt(time.Now())
		ok, err := p.Add(tx, &core.DefaultTxHasher{})
		assert.True(t, ok)
		assert.NoError(t, err, "loop %d", i)
	}
	assert.Equal(t, nTx, p.PendingCount(), "pool len %d != %d", p.PendingCount(), nTx)

	txns := p.Pending()
	for i := 0; i < p.PendingCount()-1; i += 1 {
		assert.True(t, txns[i].GetCreatedAt().Before(txns[i+1].GetCreatedAt()))
	}

	// overflow pending
	tx := core.NewTransaction([]byte("overflow"))
	tx.SetCreatedAt(time.Now())
	p.Add(tx, &core.DefaultTxHasher{})
	added, err := p.Add(tx, &core.DefaultTxHasher{})
	assert.False(t, added)
	assert.Error(t, err)

	// rollover test
	want, err := p.all.txns.Get(1)
	assert.NoError(t, err)
	p.ClearPending()
	ok, err := p.Add(core.NewTransaction([]byte("whatevs")), &core.DefaultTxHasher{})
	assert.True(t, ok)
	assert.NoError(t, err)
	got, err := p.all.First()
	assert.NoError(t, err)
	assert.Equal(t, want, got, " got %s != want %s",
		got.Hash(&core.DefaultTxHasher{}).Prefix(),
		want.Hash(&core.DefaultTxHasher{}).Prefix())
	// dedup test

	p = NewTxPool(MaxPoolDepth(2))
	for i := 0; i < 4; i += 1 {
		added, err := p.Add(core.NewTransaction([]byte("foo")), &core.DefaultTxHasher{})
		assert.NoError(t, err)
		if i == 0 {
			assert.True(t, added)
		} else {
			assert.False(t, added)
		}
	}
}
