package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

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

func newHeaderFromPrev(prev *Header) *Header {
	return &Header{
		Version:           1,
		Height:            prev.Height + 1,
		PreviousBlockHash: DefaultBlockHasher{}.Hash(prev),
		Timestamp:         uint64(time.Now().UTC().UnixNano()),
	}

}

func (h *Header) setDataHash(txns []*Transaction) (*Header, error) {
	dh, err := calculateDataHash(txns)
	if err != nil {
		return h, err
	}
	h.DataHash = dh
	return h, nil
}

type Block struct {
	*Header
	Transactions []*Transaction
	// validator and signature for verifiability of creator
	Validator crypto.PublicKey
	Signature *crypto.Signature
	// cache of header hash
	hash types.Hash
}

func NewBlock(h *Header, txns []*Transaction) *Block {

	return &Block{
		Header:       h,
		Transactions: txns,
	}
}

func NewBlockFromPrevHeader(prev *Header, txns []*Transaction) (*Block, error) {
	h, err := newHeaderFromPrev(prev).setDataHash(txns)

	if err != nil {
		return nil, err
	}
	return NewBlock(h, txns), err
}

func (b *Block) AddTransaction(tx *Transaction) {
	b.Transactions = append(b.Transactions, tx)
}

// Sign must called to finalize a block -- after all the transactions are added
func (b *Block) Sign(privKey crypto.PrivateKey) error {
	sig, err := privKey.Sign(b.Header.MustToBytes())
	if err != nil {
		return err
	}
	b.Signature = sig
	b.Validator = privKey.PublicKey()
	dh, err := calculateDataHash(b.Transactions)
	if err != nil {
		return err
	}
	b.DataHash = dh
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

	dh, err := calculateDataHash(b.Transactions)
	if err != nil {
		return nil
	}
	if dh != b.DataHash {
		return fmt.Errorf("invalid data hash. got %s != given %s", dh.String(), b.DataHash)
	}
	return nil
}

func (b *Block) Encode(encoder Encoder[*Block]) error {
	return encoder.Encode(b)
}

func (b *Block) Decode(decoder Decoder[*Block]) error {
	return decoder.Decode(b)
}

func (b *Block) Hash(hasher Hasher[*Header]) types.Hash {

	if b.hash.IsZero() {
		b.hash = hasher.Hash(b.Header)
	}

	return b.hash
}
