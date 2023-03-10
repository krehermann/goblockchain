package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
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

	hasher Hasher[*Transaction]
}

func NewTransaction(data []byte) *Transaction {
	return &Transaction{
		Data:   data,
		hasher: &DefaultTxHasher{},
	}
}

func (txn *Transaction) SetHasher(hasher Hasher[*Transaction]) {
	txn.hasher = hasher
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
func (txn *Transaction) Hash() types.Hash {
	if txn.hash.IsZero() {
		txn.hash = txn.hasher.Hash(txn)
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

func calculateDataHash(txns []*Transaction) (types.Hash, error) {
	var (
		buf  = &bytes.Buffer{}
		hash types.Hash
	)
	for _, tx := range txns {
		err := tx.Encode(NewGobTxEncoder(buf))
		if err != nil {
			return hash, err
		}
	}
	hash = sha256.Sum256(buf.Bytes())
	return hash, nil
}

// Override the default marshaler so that hash is included
func (txn *Transaction) MarshalJSON() ([]byte, error) {

	type anon struct {
		Data      []byte
		From      crypto.PublicKey
		Signature *crypto.Signature
		Hash      types.Hash
	}

	marshalMe := anon{
		Data:      txn.Data,
		From:      txn.From,
		Signature: txn.Signature,
		Hash:      txn.hash,
	}

	return json.Marshal(&marshalMe)
}
