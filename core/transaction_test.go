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
	txn.From = wrongPrivKey.PublicKey()
	assert.Error(t, txn.Verify())
}

func randomTxWithSignature(t *testing.T) *Transaction {
	tx := &Transaction{
		Data: []byte("my data"),
	}
	privKey := crypto.MustGeneratePrivateKey()
	err := tx.Sign(privKey)
	assert.NoError(t, err)
	assert.NoError(t, tx.Verify())
	return tx
}
