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
	PublicKey crypto.PublicKey
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
	txn.PublicKey = privKey.PublicKey()
	txn.Signature = sig
	return nil
}

func (txn *Transaction) Verify() error {
	if txn.Signature == nil {
		return fmt.Errorf("transactions has no signature")
	}
	ok := txn.Signature.Verify(txn.PublicKey, txn.Data)
	if !ok {
		return fmt.Errorf("invalid transaction signature")
	}
	return nil
}
