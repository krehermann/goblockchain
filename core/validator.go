package core

import "fmt"

type Validator interface {
	ValidateBlock(*Block) error
}

type BlockValidator struct {
	bc *Blockchain
}

func NewBlockValidator(bc *Blockchain) *BlockValidator {
	return &BlockValidator{
		bc: bc,
	}
}

func (v *BlockValidator) ValidateBlock(b *Block) error {
	if v.bc.HasBlockAtHeight(b.Height) {
		return fmt.Errorf("chain already has block(%d) with hash %s", b.Height, b.Hash(DefaultBlockHasher{}))
	}

	return b.Verify()

}
