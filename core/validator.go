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

	// implicitly we are only validating blocks after the genesis
	if b.Height != v.bc.Height()+1 {
		//return fmt.Errorf("chain cannot already block with height %d because does not match expect chain height %d", b.Height, v.bc.Height())
		return NewErrOutOfSync(int(b.Height) - int(v.bc.Height()))
	}

	prevHeader, err := v.bc.GetHeader(b.Height - 1)
	if err != nil {
		return nil
	}

	onChainHash := DefaultBlockHasher{}.Hash(prevHeader)
	if onChainHash != b.PreviousBlockHash {
		return fmt.Errorf("validateBlock: previous hash on chain (%s) doesn't match the value in the given block (%s)", onChainHash.String(), b.PreviousBlockHash)
	}

	err = b.Verify()
	if err != nil {
		return fmt.Errorf("invalid block: failed to verify: %w", err)
	}
	return nil

}
