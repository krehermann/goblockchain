package core

import (
	"bytes"
	"testing"
	"time"

	"github.com/krehermann/goblockchain/types"
	"github.com/stretchr/testify/assert"
)

func TestHeader_Roundtrip(t *testing.T) {
	h := &Header{
		Version:       1,
		PreviousBlock: types.RandomHash(),
		Timestamp:     uint64(time.Unix(0, 0).Unix()),
		Height:        10,
		Nonce:         1234,
	}

	buf := &bytes.Buffer{}
	assert.NoError(t, h.EncodeBinary(buf))

	hDecode := &Header{}
	assert.NoError(t, hDecode.DecodeBinary(buf))
	assert.Equal(t, h, hDecode)
}

func TestBlock_Encode_Decode(t *testing.T) {
	b := &Block{
		Header: Header{Version: 1,
			PreviousBlock: types.RandomHash(),
			Timestamp:     uint64(time.Unix(0, 0).Unix()),
			Height:        10,
			Nonce:         1234},
		Transactions: nil,
	}

	buf := &bytes.Buffer{}
	assert.NoError(t, b.EncodeBinary(buf))

	bDecode := &Block{}
	assert.NoError(t, bDecode.DecodeBinary(buf))
}

func TestBlockHash(t *testing.T) {
	b := &Block{
		Header: Header{Version: 1,
			PreviousBlock: types.RandomHash(),
			Timestamp:     uint64(time.Unix(0, 0).Unix()),
			Height:        10,
			Nonce:         1234},
		Transactions: nil,
	}
	hsh := b.Hash()
	assert.False(t, hsh.IsZero())
}
