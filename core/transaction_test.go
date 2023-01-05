package core

import (
	"bytes"
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

func TestTxnEncodeDecode(t *testing.T) {
	txn := randomTxWithSignature(t)
	buf := &bytes.Buffer{}
	assert.NoError(t, txn.Encode(NewGobTxEncoder(buf)))

	got := new(Transaction)
	assert.NoError(t, got.Decode(NewGobTxDecoder(buf)))

	assert.Equal(t, txn, got)
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
