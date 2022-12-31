package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/krehermann/goblockchain/types"
)

// wrap specific underlying keys for extensibility
type PrivateKey struct {
	key *ecdsa.PrivateKey
}

func MustGeneratePrivateKey() PrivateKey {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("error creating private key"))
	}
	return PrivateKey{
		key: key,
	}
}

func (k *PrivateKey) PublicKey() PublicKey {
	return PublicKey{
		key: &k.key.PublicKey,
	}
}

func (k PrivateKey) Sign(data []byte) (*Signature, error) {
	r, s, err := ecdsa.Sign(rand.Reader, k.key, data)
	if err != nil {
		return nil, err
	}

	return &Signature{
		r: r,
		s: s}, nil
}

type PublicKey struct {
	key *ecdsa.PublicKey
}

func (k PublicKey) ToSlice() []byte {
	return elliptic.MarshalCompressed(k.key, k.key.X, k.key.Y)

}

func (k PublicKey) Address() types.Address {
	h := sha256.Sum256(k.ToSlice())

	end := h[len(h)-20:]
	return types.MustAddressFromBytes(end)
}

type Signature struct {
	r *big.Int
	s *big.Int
}

func (s *Signature) Verify(pubKey PublicKey, data []byte) bool {
	return ecdsa.Verify(pubKey.key, data, s.r, s.s)
}
