package core

import (
	"fmt"
	"sync"

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

type Blockchain struct {
	store Storager

	headerMu sync.RWMutex
	headers  []*Header

	validator Validator
	logger    *zap.Logger

	// TODO make State interface
	contractState *vm.State
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
		headers:       []*Header{},
		store:         NewMemStore(),
		logger:        zap.L().Named("blockchain"),
		contractState: vm.NewState(),
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
				tx.Hash(&DefaultTxHasher{}).Prefix(),
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
	// height doesn't count the last header
	bc.headerMu.RLock()
	defer bc.headerMu.RUnlock()

	return uint32(len(bc.headers) - 1)
}

func (bc *Blockchain) HasBlockAtHeight(h uint32) bool {
	return h <= bc.Height()
}

func (bc *Blockchain) GetHeader(height uint32) (*Header, error) {

	if height > bc.Height() {
		return &Header{}, fmt.Errorf("cannot get header for block height %d: out of range %d", height, bc.Height())
	}

	bc.headerMu.RLock()
	defer bc.headerMu.RUnlock()

	return bc.headers[height], nil
}

func (bc *Blockchain) addGensisBlock(b *Block) error {
	// scope the lock
	err := func() error {
		bc.headerMu.RLock()
		defer bc.headerMu.RUnlock()

		if len(bc.headers) > 0 {
			return fmt.Errorf("addGenesisBlock: refusing to add genesis block to non-0 len chain")
		}
		return nil
	}()

	if err != nil {
		return err
	}
	return bc.persistBlock(b)
}

func (bc *Blockchain) persistBlock(b *Block) error {
	bc.headerMu.Lock()
	bc.headers = append(bc.headers, b.Header)
	bc.headerMu.Unlock()

	bc.logger.Info("added block to chain",
		zap.Int("height", int(bc.Height())),
		zap.String("hash", b.Hash(DefaultBlockHasher{}).String()),
		zap.Any("tx len", len(b.Transactions)),
	)
	return bc.store.Put(b)

}
