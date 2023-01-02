package core

import (
	"testing"
	"time"

	"github.com/krehermann/goblockchain/crypto"
	"github.com/krehermann/goblockchain/types"
	"github.com/stretchr/testify/assert"
)

func randomBlock(height uint32) *Block {
	b := &Block{
		Header: &Header{
			Version:           1,
			PreviousBlockHash: types.RandomHash(),
			Timestamp:         uint64(time.Now().Unix()),
			Height:            height,
		},
		Transactions: []Transaction{
			Transaction{
				Data: []byte("tengo hambre"),
			},
		},
	}

	return b
}

func randomBlockWithSignature(t *testing.T, height uint32) *Block {
	privKey := crypto.MustGeneratePrivateKey()
	bc := randomBlock(height)
	assert.NoError(t, bc.Sign(privKey))
	return bc
}

func TestBlock_SignAndVerify(t *testing.T) {
	privKey := crypto.MustGeneratePrivateKey()
	b := randomBlock(0)

	assert.NoError(t, b.Sign(privKey))
	assert.NoError(t, b.Verify())

	// something is wrong about the signatures
	// leave this failing test for now
	x := randomBlock(21)
	assert.NoError(t, x.Sign(privKey))
	assert.NoError(t, x.Verify())
	assert.False(t, x.Signature.Verify(x.Validator, b.MustHeaderData()))

	// change the block validator
	wrongPrivKey := crypto.MustGeneratePrivateKey()
	wrongPubKey := wrongPrivKey.PublicKey()
	b.Validator = wrongPubKey
	assert.Error(t, b.Verify())

}
