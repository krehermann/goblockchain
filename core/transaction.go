package core

import (
	"fmt"
	"io"

	"github.com/krehermann/goblockchain/crypto"
)

type Transaction struct {
	// make it work
	// make it better
	// make it fast

	Data      []byte
	From      crypto.PublicKey
	Signature *crypto.Signature
}

func (txn *Transaction) EncodeBinary(w io.Writer) error {
	return nil
}

func (txn *Transaction) DecodeBinary(r io.Reader) error {
	return nil
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
