package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestBlockChain(t *testing.T, genesis *Block) *Blockchain {
	bc, err := NewBlockchain(genesis)

	assert.NoError(t, err)
	assert.NotNil(t, bc)
	return bc
}

func TestBlockchina(t *testing.T) {
	bc := newTestBlockChain(t, randomBlock(0))
	assert.Equal(t, uint32(0), bc.Height())
	assert.NotNil(t, bc.validator)
}

func TestHasBlock(t *testing.T) {
	bc := newTestBlockChain(t, randomBlock(0))

	assert.True(t, bc.HasBlockAtHeight(0))
}

func TestAddBlock(t *testing.T) {
	bc := newTestBlockChain(t, randomBlock(0))

	nBlocks := uint32(100)
	for i := uint32(1); i <= nBlocks; i += 1 {
		assert.NoError(t, bc.AddBlock(randomBlockWithSignature(t, i)))
		assert.True(t, bc.HasBlockAtHeight(i))
	}

	assert.Equal(t, nBlocks, bc.Height())

	assert.Error(t, bc.AddBlock(randomBlock(42)))
}
