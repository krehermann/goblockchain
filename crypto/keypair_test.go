package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyPair_Sign_Verify(t *testing.T) {
	privKey := MustGeneratePrivateKey()
	pubKey := privKey.PublicKey()
	msg := []byte("hello, friend")

	sig, err := privKey.Sign(msg)
	assert.NoError(t, err)
	assert.True(t, sig.Verify(pubKey, msg))
	assert.False(t, sig.Verify(pubKey, []byte("different data")))
	wrongPrivKey := MustGeneratePrivateKey()
	wrongPubKey := wrongPrivKey.PublicKey()
	assert.False(t, sig.Verify(wrongPubKey, msg))
}
