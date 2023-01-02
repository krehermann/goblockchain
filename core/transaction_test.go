package core

import (
	"testing"

	"github.com/krehermann/goblockchain/crypto"
	"github.com/stretchr/testify/assert"
)

func TestTxnSign(t *testing.T) {
	privKey := crypto.MustGeneratePrivateKey()
	d := []byte("hi")
	txn := &Transaction{
		Data: d,
	}

	err := txn.Sign(privKey)
	assert.NoError(t, err)
	assert.NoError(t, txn.Verify())

	txn.Data = []byte("altered")
	assert.Error(t, txn.Verify())
	txn.Data = d
	assert.NoError(t, txn.Verify())

	wrongPrivKey := crypto.MustGeneratePrivateKey()
	// change the txn and check that verify fails
	txn.PublicKey = wrongPrivKey.PublicKey()
	assert.Error(t, txn.Verify())
}
