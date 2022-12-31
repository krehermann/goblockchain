package types

import (
	"encoding/hex"
	"fmt"
)

const AddressLen = 20

type Address [AddressLen]uint8

func (a Address) ToSlice() []byte {
	out := make([]byte, AddressLen)
	for i := range a {
		out[i] = a[i]
	}
	return out
}

func MustAddressFromBytes(b []byte) Address {
	if len(b) < AddressLen {
		panic(fmt.Sprintf("input byte slice less than required len %d < %d", len(b), AddressLen))
	}
	out := Address{}
	for i := range b {
		out[i] = b[i]
	}
	return out
}

func (a Address) String() string {
	return hex.EncodeToString(a.ToSlice())
}
