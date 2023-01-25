package core

import (
	"fmt"
	"sync"

	"github.com/krehermann/goblockchain/types"
	"github.com/krehermann/goblockchain/vm"
	"go.uber.org/zap"
)

// start with a struct, later move to interface

// Blockchain is essentially a big state machine
// each transaction can cause a transition

type ErrOutOfSync struct {
	//err error
	Lag int
}

func NewErrOutOfSync(lag int) *ErrOutOfSync {
	return &ErrOutOfSync{
		Lag: lag,
	}
}

func (eos *ErrOutOfSync) Error() string {
	return fmt.Sprintf("behind by %d blocks", eos.Lag)
}

type headerStore struct {
	m    sync.RWMutex
	data []*Header
	idx  map[types.Hash]int
}

func newheaderStore() *headerStore {
	return &headerStore{
		m:    sync.RWMutex{},
		data: make([]*Header, 0),
		idx:  make(map[types.Hash]int),
	}
}

func (hs *headerStore) put(blockId types.Hash, hdr *Header) error {
	hs.m.RLock()
	got, exists := hs.idx[blockId]
	hs.m.RUnlock()
	if exists {
		return fmt.Errorf("header (%+v) already exists %+v", hdr, got)
	}

	hs.m.Lock()
	defer hs.m.Unlock()
	hs.data = append(hs.data, hdr)
	hs.idx[blockId] = len(hs.data) - 1
	return nil
}

func (hs *headerStore) empty() bool {
	hs.m.RLock()
	defer hs.m.RUnlock()
	return len(hs.data) == 0
}

func (hs *headerStore) height() uint32 {
	hs.m.RLock()
	defer hs.m.RUnlock()

	return uint32(len(hs.data) - 1)
}

func (hs *headerStore) getAt(height uint32) (*Header, error) {

	if height > hs.height() {
		return nil, fmt.Errorf("out of range. request height (%d) > %d", height, hs.height())
	}

	hs.m.RLock()
	defer hs.m.RUnlock()
	return hs.data[height], nil

}

func (hs *headerStore) getHash(hash types.Hash) (*Header, error) {
	hs.m.RLock()
	ix, exists := hs.idx[hash]
	hs.m.RUnlock()

	if !exists {
		return nil, fmt.Errorf("header does not exist for hash %s", hash.String())
	}

	hs.m.Lock()
	defer hs.m.Unlock()
	return hs.data[ix], nil
}

type Blockchain struct {
	//store Storager

	store   GenericStorager[*Header, *Block]
	txStore GenericStorager[types.Hash, *Transaction]
	headers *headerStore

	validator Validator
	logger    *zap.Logger

	// TODO make State interface
	contractState *vm.State

	hasher   Hasher[*Header]
	txHasher Hasher[*Transaction]
}

type BlockchainOpt func(bc *Blockchain) *Blockchain

func WithLogger(l *zap.Logger) BlockchainOpt {
	return func(bc *Blockchain) *Blockchain {
		bc.logger = l.Named("blockchain")
		return bc
	}
}

func NewBlockchain(genesis *Block, opts ...BlockchainOpt) (*Blockchain, error) {
	bc := &Blockchain{
		headers:       newheaderStore(),
		store:         NewGenericMemStore[*Header, *Block](),
		txStore:       NewGenericMemStore[types.Hash, *Transaction](),
		logger:        zap.L().Named("blockchain"),
		contractState: vm.NewState(),
		hasher:        DefaultBlockHasher{},
		txHasher:      &DefaultTxHasher{},
	}

	// this is a bit weird. need to be able to configure the validator
	// left for later
	bc.WithValidator(NewBlockValidator(bc))
	for _, opt := range opts {
		bc = opt(bc)
	}

	err := bc.addGensisBlock(genesis)

	return bc, err
}

func (bc *Blockchain) AddBlock(b *Block) error {
	bc.logger.Debug("Add block",
		zap.Any("header", b.Header),
	)
	err := bc.validator.ValidateBlock(b)
	if err != nil {
		return err
		//return fmt.Errorf("add block: invalid block: %w", err)
	}

	// run vm code
	// KISS for now -- just make a vm

	for _, tx := range b.Transactions {
		vm := vm.NewVM(tx.Data,
			vm.ContractStateOpt(bc.contractState),
			vm.LoggerOpt(bc.logger))
		err := vm.Run()
		if err != nil {
			return fmt.Errorf("add block: vm failed to run tx %s, %w",
				tx.Hash().Prefix(),
				err)
		}
		// result is the last thing on the stack
		val, _ := vm.Stack.Peek()
		bc.logger.Debug("vm result",
			zap.Any("val", val),
		)
	}

	return bc.persistBlock(b)
}

func (bc *Blockchain) WithValidator(v Validator) {
	bc.validator = v
}

// thread safe
func (bc *Blockchain) Height() uint32 {
	return bc.headers.height()
}

func (bc *Blockchain) HasBlockAtHeight(h uint32) bool {
	return h <= bc.Height()
}

func (bc *Blockchain) GetHeader(height uint32) (*Header, error) {
	return bc.headers.getAt(height)
}

func (bc *Blockchain) GetBlockAt(height uint32) (*Block, error) {
	hdr, err := bc.GetHeader(height)
	if err != nil {
		return nil, err
	}

	return bc.store.Get(hdr)
}

func (bc *Blockchain) GetBlockHash(h types.Hash) (*Block, error) {
	hdr, err := bc.headers.getHash(h)
	if err != nil {
		return nil, err
	}
	return bc.store.Get(hdr)
}

func (bc *Blockchain) GetTransaction(h types.Hash) (*Transaction, error) {
	return bc.txStore.Get(h)
}

func (bc *Blockchain) addGensisBlock(b *Block) error {
	// scope the lock

	if !bc.headers.empty() {
		return fmt.Errorf("addGenesisBlock: refusing to add genesis block to non-0 len chain")
	}
	return bc.persistBlock(b)
}

func (bc *Blockchain) persistBlock(b *Block) error {
	bh := b.Hash(bc.hasher)
	err := bc.headers.put(bh, b.Header)
	if err != nil {
		return err
	}

	bc.logger.Info("added block to chain",
		zap.Int("height", int(bc.Height())),
		zap.String("hash", b.Hash(bc.hasher).String()),
		zap.Any("tx len", len(b.Transactions)),
	)
	err = bc.store.Put(b.Header, b)
	if err != nil {
		return nil
	}

	for _, txn := range b.Transactions {
		txn.SetHasher(bc.txHasher)
		err := bc.txStore.Put(txn.Hash(), txn)
		if err != nil {
			return err
		}
	}
	return nil
}
