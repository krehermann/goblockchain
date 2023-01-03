package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/krehermann/goblockchain/crypto"
	"github.com/krehermann/goblockchain/types"
)

// header has all the info needed to store in the merkle tree
type Header struct {
	Version           uint32
	DataHash          types.Hash
	PreviousBlockHash types.Hash
	Timestamp         uint64
	Height            uint32
}

func (h *Header) MustToBytes() []byte {
	//if b.headerData == nil {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	err := enc.Encode(h)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

type Block struct {
	*Header
	Transactions []Transaction
	// validator and signature for verifiability of creator
	Validator crypto.PublicKey
	Signature *crypto.Signature
	// cache of header hash
	hash types.Hash
}

func NewBlock(h *Header, txns []Transaction) *Block {
	return &Block{
		Header:       h,
		Transactions: txns,
	}
}

func (b *Block) AddTransaction(tx *Transaction) {
	b.Transactions = append(b.Transactions, *tx)
}

func (b *Block) Sign(privKey crypto.PrivateKey) error {
	sig, err := privKey.Sign(b.Header.MustToBytes())
	if err != nil {
		return err
	}
	b.Signature = sig
	b.Validator = privKey.PublicKey()
	return nil
}

func (b *Block) Verify() error {
	if b.Signature == nil {
		return fmt.Errorf("block has no signature")
	}
	if !b.Signature.Verify(b.Validator, b.Header.MustToBytes()) {
		return fmt.Errorf("invalid block signature")
	}

	for i, tx := range b.Transactions {
		err := tx.Verify()
		if err != nil {
			return fmt.Errorf("block verify failed at transaction %d: %w", i, err)
		}
	}
	return nil
}

func (b *Block) Encode(w io.Writer, encoder Encoder[*Block]) error {
	return encoder.Encode(w, b)
}

func (b *Block) Decode(r io.Reader, decoder Decoder[*Block]) error {
	return decoder.Decode(r, b)
}

func (b *Block) Hash(hasher Hasher[*Header]) types.Hash {

	if b.hash.IsZero() {
		b.hash = hasher.Hash(b.Header)
	}

	return b.hash
}
