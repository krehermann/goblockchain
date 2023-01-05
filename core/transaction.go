package core

import (
	"fmt"
	"time"

	"github.com/krehermann/goblockchain/crypto"
	"github.com/krehermann/goblockchain/types"
)

type Transaction struct {
	// make it work
	// make it better
	// make it fast

	Data      []byte
	From      crypto.PublicKey
	Signature *crypto.Signature

	// note that these private field are not encoding
	// cached hash
	hash types.Hash
	// keep track for ordering
	createdAt time.Time
}

func NewTransaction(data []byte) *Transaction {
	return &Transaction{
		Data: data,
	}
}

func (txn *Transaction) SetCreatedAt(t time.Time) *Transaction {
	txn.createdAt = t
	return txn
}

func (txn *Transaction) GetCreatedAt() time.Time {
	if txn.createdAt.IsZero() {
		txn.createdAt = time.Now()
	}
	return txn.createdAt
}

func (txn *Transaction) Encode(enc Encoder[*Transaction]) error {
	return enc.Encode(txn)
}

func (txn *Transaction) Decode(dec Decoder[*Transaction]) error {
	return dec.Decode(txn)
}

// Hash caches the first call. Subsequent changes to input Hasher have no effect
func (txn *Transaction) Hash(hasher Hasher[*Transaction]) types.Hash {
	if txn.hash.IsZero() {
		txn.hash = hasher.Hash(txn)
	}
	return txn.hash
}

func (txn *Transaction) Sign(privKey crypto.PrivateKey) error {
	sig, err := privKey.Sign(txn.Data)
	if err != nil {
		return err
	}
	if sig == nil {
		panic("nil signature in transaction after signing")
	}
	txn.From = privKey.PublicKey()
	txn.Signature = sig
	return nil
}

func (txn *Transaction) Verify() error {
	if txn.Signature == nil {
		return fmt.Errorf("transaction has no signature")
	}
	ok := txn.Signature.Verify(txn.From, txn.Data)
	if !ok {
		return fmt.Errorf("invalid transaction signature")
	}
	return nil
}
