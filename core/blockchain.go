package core

import (
	"fmt"
	"sync"
)

// start with a struct, later move to interface

// Blockchain is essentially a big state machine
// each transaction can cause a transition

type Blockchain struct {
	store Storager

	headerMu sync.RWMutex
	headers  []*Header

	validator Validator
}

func NewBlockchain(genesis *Block) (*Blockchain, error) {
	bc := &Blockchain{
		headers: []*Header{},
		store:   NewMemStore(),
	}

	// this is a bit weird. need to be able to configure the validator
	// left for later
	bc.WithValidator(NewBlockValidator(bc))
	err := bc.addGensisBlock(genesis)

	return bc, err
}

func (bc *Blockchain) AddBlock(b *Block) error {
	err := bc.validator.ValidateBlock(b)
	if err != nil {
		return fmt.Errorf("add block: invalid block: %w", err)
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

	return bc.store.Put(b)

}
