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

func (k PrivateKey) IsZero() bool {
	return k.key == nil
}

func (k PrivateKey) PublicKey() PublicKey {
	return elliptic.MarshalCompressed(k.key.PublicKey,
		k.key.PublicKey.X,
		k.key.PublicKey.Y)
}

func (k PrivateKey) Sign(data []byte) (*Signature, error) {
	r, s, err := ecdsa.Sign(rand.Reader, k.key, data)
	if err != nil {
		return nil, err
	}

	return &Signature{
		S: s,
		R: r}, nil
}

type PublicKey []byte

func (k PublicKey) ToSlice() []byte {
	return k

}

func (k PublicKey) Address() types.Address {
	h := sha256.Sum256(k.ToSlice())

	end := h[len(h)-20:]
	return types.MustAddressFromBytes(end)
}

type Signature struct {
	R *big.Int
	S *big.Int
}

func (s *Signature) Verify(pubKey PublicKey, data []byte) bool {
	x, y := elliptic.UnmarshalCompressed(elliptic.P256(), pubKey)

	key := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	return ecdsa.Verify(key, data, s.R, s.S)
}
