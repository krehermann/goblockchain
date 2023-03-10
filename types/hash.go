package types

import (
	"encoding/hex"
	"fmt"
	"math/rand"
)

const HASH_BYTE_LEN = 32

type Hash [HASH_BYTE_LEN]uint8

func HashFromBytes(b []byte) Hash {
	if len(b) != HASH_BYTE_LEN {
		panic(fmt.Sprintf("given byte slice len %d but must be HASH_BYTE_LEN", len(b)))
	}

	var val [HASH_BYTE_LEN]uint8
	for i := range b {
		val[i] = b[i]
	}

	return Hash(val)
}

func RandomBytes(size int) []byte {
	val := make([]byte, size)
	rand.Read(val)
	return val
}

func RandomHash() Hash {
	return HashFromBytes(RandomBytes(HASH_BYTE_LEN))
}

func (h Hash) IsZero() bool {
	for _, b := range h {
		if b != 0 {
			return false
		}
	}
	return true
}

func (h Hash) ToSlice() []byte {
	out := make([]byte, 32)
	for i := range h {
		out[i] = h[i]
	}
	return out
}

func (h Hash) String() string {
	return hex.EncodeToString(h.ToSlice())
}

func (h Hash) Prefix() string {
	r := []rune(h.String())
	return string(r[:8])
}

func (h Hash) MarshalJSON() ([]byte, error) {

	return []byte(
		fmt.Sprintf("\"%s\"",
			h.String())), nil
}

func HashFromHex(hx string) (Hash, error) {
	b, err := hex.DecodeString(hx)
	if err != nil {
		return Hash{}, err
	}
	return HashFromBytes(b), nil
}

/*
func (h Hash) UnmarshalJSON([]byte) ([]byte,error) {
	return []byte(h.String()), nil
}
*/
