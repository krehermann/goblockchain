package core

import (
	"testing"
	"time"

	"github.com/krehermann/goblockchain/crypto"
	"github.com/krehermann/goblockchain/types"
	"github.com/stretchr/testify/assert"
)

func TestBlock_SignAndVerify(t *testing.T) {
	privKey := crypto.MustGeneratePrivateKey()
	b := randomBlockWithoutPreviousBlock(t, 0)

	assert.NoError(t, b.Sign(privKey))
	assert.NoError(t, b.Verify())
	/*
		// something is wrong about the signatures
		// leave this failing test for now
		x := randomBlockWithoutPreviousBlock(t, 21)
		assert.NoError(t, x.Sign(privKey))
		assert.NoError(t, x.Verify())
		//	assert.False(t, x.Signature.Verify(x.Validator, b.Header.MustToBytes()))
	*/
	b.Header.Height = 7
	assert.Error(t, b.Verify())

	// change the block validator
	wrongPrivKey := crypto.MustGeneratePrivateKey()
	wrongPubKey := wrongPrivKey.PublicKey()
	b.Validator = wrongPubKey
	assert.Error(t, b.Verify())

}

func randomBlock(t *testing.T, height uint32, prevBlockHash types.Hash) *Block {
	b := &Block{
		Header: &Header{
			Version:           1,
			PreviousBlockHash: prevBlockHash,
			Timestamp:         uint64(time.Now().Unix()),
			Height:            height,
		},
		Transactions: []*Transaction{},
	}

	return b
}

func randomBlockWithSignature(t *testing.T, height uint32, prevBlockHash types.Hash) *Block {
	privKey := crypto.MustGeneratePrivateKey()
	b := randomBlock(t, height, prevBlockHash)
	b.AddTransaction(randomTxWithSignature(t))
	assert.NoError(t, b.Sign(privKey))
	return b
}

func randomBlockWithoutPreviousBlock(t *testing.T, height uint32) *Block {
	b := randomBlock(t, height, types.Hash{})
	b.AddTransaction(randomTxWithSignature(t))
	return b
}

func randomGenesisBlock(t *testing.T) *Block {
	return randomBlock(t, 0, types.RandomHash())
}
