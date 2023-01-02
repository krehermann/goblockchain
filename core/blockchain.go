package core

import "fmt"

// start with a struct, later move to interface
type Blockchain struct {
	store     Storager
	headers   []*Header
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

func (bc *Blockchain) Height() uint32 {
	// height doesn't count the last header
	return uint32(len(bc.headers) - 1)
}

func (bc *Blockchain) HasBlockAtHeight(h uint32) bool {
	return h <= bc.Height()
}

func (bc *Blockchain) addGensisBlock(b *Block) error {
	if len(bc.headers) > 0 {
		return fmt.Errorf("addGenesisBlock: refusing to add genesis block to non-0 len chain")
	}
	return bc.persistBlock(b)
}

func (bc *Blockchain) persistBlock(b *Block) error {
	bc.headers = append(bc.headers, b.Header)
	return bc.store.Put(b)

}
