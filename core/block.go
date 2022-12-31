package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"io"

	"github.com/krehermann/goblockchain/types"
)

// header has all the info needed to store in the merkle tree
type Header struct {
	Version       uint32
	PreviousBlock types.Hash
	Timestamp     uint64
	Height        uint32
	Nonce         uint64
}

func (h *Header) DecodeBinary(r io.Reader) error {
	err := binary.Read(r, binary.LittleEndian, &h.Version)
	if err != nil {
		return err
	}
	err = binary.Read(r, binary.LittleEndian, &h.PreviousBlock)
	if err != nil {
		return err
	}
	err = binary.Read(r, binary.LittleEndian, &h.Timestamp)
	if err != nil {
		return err
	}
	err = binary.Read(r, binary.LittleEndian, &h.Height)
	if err != nil {
		return err
	}
	return binary.Read(r, binary.LittleEndian, &h.Nonce)

}

// this is very generic. the version is very important because in a full
// impl is dictated how to decode the data
func (h *Header) EncodeBinary(w io.Writer) error {
	err := binary.Write(w, binary.LittleEndian, &h.Version)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.LittleEndian, &h.PreviousBlock)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.LittleEndian, &h.Timestamp)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.LittleEndian, &h.Height)
	if err != nil {
		return err
	}
	return binary.Write(w, binary.LittleEndian, &h.Nonce)

}

type Block struct {
	Header
	Transactions []Transaction
	// cache of header hash
	hash types.Hash
}

func (b *Block) EncodeBinary(w io.Writer) error {
	err := b.Header.EncodeBinary(w)
	if err != nil {
		return err
	}

	for _, tx := range b.Transactions {
		err = tx.EncodeBinary(w)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Block) DecodeBinary(r io.Reader) error {
	err := b.Header.DecodeBinary(r)
	if err != nil {
		return err
	}

	for _, tx := range b.Transactions {
		err = tx.DecodeBinary(r)
		if err != nil {
			return err
		}
	}
	return nil

}

func (b *Block) Hash() types.Hash {
	buf := &bytes.Buffer{}
	b.Header.EncodeBinary(buf)

	b.hash = types.Hash(sha256.Sum256(buf.Bytes()))

	return b.hash
}
