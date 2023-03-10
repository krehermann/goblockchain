package core

import (
	"math/rand"
	"testing"

	"github.com/krehermann/goblockchain/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockchian(t *testing.T) {
	bc := newTestBlockChain(t, randomGenesisBlock(t))
	assert.Equal(t, uint32(0), bc.Height())
	assert.NotNil(t, bc.validator)
}

func TestHasBlock(t *testing.T) {
	bc := newTestBlockChain(t, randomGenesisBlock(t))

	assert.True(t, bc.HasBlockAtHeight(0))
	assert.False(t, bc.HasBlockAtHeight(1))
	assert.False(t, bc.HasBlockAtHeight(100))

}

func TestAddBlock(t *testing.T) {
	bc := newTestBlockChain(t, randomGenesisBlock(t))

	expectedBlocks := make(map[uint32]*Block)

	nBlocks := uint32(100)
	for i := uint32(1); i <= nBlocks; i += 1 {
		prevBlockHash := getPrevBlockHash(t, bc, i)
		b := randomBlockWithSignature(t, i, prevBlockHash)
		require.NoError(t, bc.AddBlock(b), "adding block %d", i)
		assert.True(t, bc.HasBlockAtHeight(i))

		got, err := bc.GetBlockAt(i)
		assert.NoError(t, err)
		assert.Equal(t, b, got)

		// pick random height and make sure we can get the block
		if i > 1 {
			prevHeight := rand.Intn(int(i)-1) + 1
			gotPrev, err := bc.GetBlockAt(uint32(prevHeight))
			assert.NoError(t, err)
			assert.Equal(t, expectedBlocks[uint32(prevHeight)], gotPrev, "iteration %d, prev height %d", i, prevHeight)
		}
		// add to the expected list
		expectedBlocks[i] = b
	}

	assert.Equal(t, nBlocks, bc.Height())
	// already exists
	prev := getPrevBlockHash(t, bc, 42)
	assert.Error(t, bc.AddBlock(randomBlockWithSignature(t, 42, prev)))

	// too high
	assert.Error(t, bc.AddBlock(randomBlockWithSignature(t, nBlocks*2, types.Hash{})))

}

func TestGetHeader(t *testing.T) {
	bc := newTestBlockChain(t, randomGenesisBlock(t))

	h, err := bc.GetHeader(0)
	assert.NoError(t, err)
	assert.NotNil(t, h)

	nBlocks := uint32(5)
	for i := uint32(1); i <= nBlocks; i += 1 {
		prevBlockHash := getPrevBlockHash(t, bc, i)
		b := randomBlockWithSignature(t, i, prevBlockHash)
		assert.NoError(t, bc.AddBlock(b))
		currHeader, err := bc.GetHeader(i)

		assert.NoError(t, err)
		assert.NotNil(t, currHeader)
		assert.Equal(t, b.Header, currHeader)

	}

}

func newTestBlockChain(t *testing.T, genesis *Block) *Blockchain {
	bc, err := NewBlockchain(genesis)

	assert.NoError(t, err)
	assert.NotNil(t, bc)
	return bc
}

func getPrevBlockHash(t *testing.T, bc *Blockchain, height uint32) types.Hash {
	prevHeader, err := bc.GetHeader(height - 1)
	assert.NoError(t, err)
	// this is used in multiple places. should abstract
	// no sure how to do that with the right layer of the Hasher
	onChainHash := DefaultBlockHasher{}.Hash(prevHeader)
	return onChainHash
}
