package server

import (
	"fmt"
	"sync"

	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/types"
	"go.uber.org/zap"
)

type TxPool struct {
	lock sync.RWMutex
	//transactions map[types.Hash]*core.Transaction
	all     *TxOrderedMap
	pending *TxOrderedMap
	logger  *zap.Logger
	max     int
	hasher  core.Hasher[*core.Transaction]
}

type TxPoolOpt func(*TxPool) *TxPool

func WithLogger(l *zap.Logger) TxPoolOpt {
	return func(p *TxPool) *TxPool {
		if l != nil {
			p.logger = l
		}
		return p
	}
}

func WithHasher(h core.Hasher[*core.Transaction]) TxPoolOpt {
	return func(p *TxPool) *TxPool {
		p.hasher = h
		return p
	}
}
func MaxPoolDepth(m int) TxPoolOpt {
	return func(p *TxPool) *TxPool {
		p.max = m
		return p
	}
}

func NewTxPool(opts ...TxPoolOpt) *TxPool {
	p := &TxPool{
		all:     NewTxOrderedMap(),
		pending: NewTxOrderedMap(),

		logger: zap.L(),
		max:    1000,
		hasher: &core.DefaultTxHasher{},
	}
	for _, opt := range opts {
		p = opt(p)
	}
	p.logger = p.logger.Named("txpool")
	return p
}

// transactions need to ordered. a simple way to do this FIFO
// on the otherhand eth uses a priority that costs gas
func (p *TxPool) Pending() []*core.Transaction {
	return p.pending.txns.Slice()
}

func (p *TxPool) PendingCount() int {
	return p.pending.Len()
}

func (p *TxPool) ClearPending() {
	p.pending.Clear()
}

// Add adds transaction of the mempool. returns any error and returns true if the
// add was ok
func (p *TxPool) Add(tx *core.Transaction) (bool, error) {
	if p.pending.Len() == p.max {
		return false, fmt.Errorf("no more space for pending transactions")
	}

	tx.SetHasher(p.hasher)
	txHash := tx.Hash()
	if p.Contains(txHash) {
		return false, nil
	}
	if p.all.Len() == p.max {
		// prune the oldest tx
		first, err := p.all.First()
		if err != nil {
			return false, err
		}
		p.logger.Debug("rolling over pool",
			zap.String("remove", first.Hash().Prefix()),
			zap.String("add", tx.Hash().Prefix()),
		)

		p.all.Remove(first.Hash())
	}

	var rollback error
	// transactional across the maps
	defer func() {
		if rollback != nil {
			p.all.Remove(txHash)
			p.pending.Remove(txHash)
		}
	}()

	rollback = p.all.Add(tx)
	if rollback != nil {
		return false, rollback
	}
	rollback = p.pending.Add(tx)
	if rollback != nil {
		return false, rollback
	}
	return true, nil

}

func (p *TxPool) Contains(h types.Hash) bool {
	return p.pending.Contains(h)
}

type TxOrderedMap struct {
	lock   sync.RWMutex
	lookup map[types.Hash]*core.Transaction
	txns   *types.List[*core.Transaction]
}

func NewTxOrderedMap() *TxOrderedMap {
	return &TxOrderedMap{
		lock:   sync.RWMutex{},
		lookup: make(map[types.Hash]*core.Transaction),
		txns:   types.NewList[*core.Transaction](),
	}
}

func (tm *TxOrderedMap) Len() int {
	tm.lock.RLock()
	defer tm.lock.RUnlock()
	return tm.txns.Len()
}

func (tm *TxOrderedMap) Contains(h types.Hash) bool {
	tm.lock.RLock()
	defer tm.lock.RUnlock()
	_, exists := tm.lookup[h]
	return exists
}

func (tm *TxOrderedMap) Add(tx *core.Transaction) error {
	h := tx.Hash()
	tm.lock.Lock()
	defer tm.lock.Unlock()
	tm.lookup[h] = tx
	tm.txns.Append(tx)
	return nil
}

func (tm *TxOrderedMap) First() (*core.Transaction, error) {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	return tm.txns.Get(0)
}

func (tm *TxOrderedMap) Get(h types.Hash) *core.Transaction {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	return tm.lookup[h]
}

func (tm *TxOrderedMap) Remove(h types.Hash) {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	val, exists := tm.lookup[h]
	if !exists {
		return
	}
	tm.txns.Delete(val)
	delete(tm.lookup, h)
}

func (tm *TxOrderedMap) Clear() {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	tm.lookup = make(map[types.Hash]*core.Transaction)
	tm.txns.Clear()
}
