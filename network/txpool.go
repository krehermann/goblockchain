package network

import (
	"sort"
	"sync"

	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/types"
)

type TxMapSorter struct {
	txns []*core.Transaction
}

func NewTxMapSorter(txMap map[types.Hash]*core.Transaction) *TxMapSorter {
	txns := make([]*core.Transaction, len(txMap))

	i := 0
	for _, tx := range txMap {
		txns[i] = tx
		i += 1
	}
	out := &TxMapSorter{
		txns: txns,
	}

	sort.Sort(out)
	return out

}
func (s *TxMapSorter) Get() []*core.Transaction {
	return s.txns
}
func (s *TxMapSorter) Len() int { return len(s.txns) }
func (s *TxMapSorter) Swap(i, j int) {
	s.txns[i], s.txns[j] = s.txns[j], s.txns[i]
}
func (s *TxMapSorter) Less(i, j int) bool {
	return s.txns[i].GetCreatedAt().Before(s.txns[j].GetCreatedAt())
}

type TxPool struct {
	lock         sync.RWMutex
	transactions map[types.Hash]*core.Transaction
}

func NewTxPool() *TxPool {
	return &TxPool{
		transactions: make(map[types.Hash]*core.Transaction),
	}
}

// transactions need to ordered. a simple way to do this FIFO
// on the otherhand eth uses a priority that costs gas
func (p *TxPool) Transactions() []*core.Transaction {
	return NewTxMapSorter(p.transactions).Get()
}

func (p *TxPool) Len() int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return len(p.transactions)
}

// Add adds transaction of the mempool. returns any error and returns true if the
// add was ok
func (p *TxPool) Add(tx *core.Transaction, hasher core.Hasher[*core.Transaction]) (bool, error) {

	// we are using a map. in this case the check here is redundant b/c
	// inserting into the map is idempotent.
	hash := tx.Hash(hasher)
	// it's normal to see the same transaction multiple times
	if p.Has(hash) {
		return false, nil
	}

	p.lock.Lock()
	defer p.lock.Unlock()
	p.transactions[hash] = tx
	return true, nil
}

func (p *TxPool) Has(h types.Hash) bool {

	p.lock.RLock()
	defer p.lock.RUnlock()

	_, exists := p.transactions[h]
	return exists
}

func (p *TxPool) Flush() {
	// reset the pool
	p.lock.Lock()
	p.transactions = make(map[types.Hash]*core.Transaction)
	p.lock.Unlock()
}
