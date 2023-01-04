package core

import (
	"crypto/sha256"

	"github.com/krehermann/goblockchain/types"
)

type Hasher[T any] interface {
	Hash(T) types.Hash
}

type DefaultBlockHasher struct{}

func (dh DefaultBlockHasher) Hash(h *Header) types.Hash {

	hash := sha256.Sum256(h.MustToBytes())
	return types.Hash(hash)
}

type DefaultTxHasher struct{}

func (txh *DefaultTxHasher) Hash(tx *Transaction) types.Hash {
	h := sha256.Sum256(tx.Data)
	return types.Hash(h)

}
