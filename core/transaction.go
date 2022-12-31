package core

import "io"

type Transaction struct {
	// make it work
	// make it better
	// make it fast

	Data []byte
}

func (txn *Transaction) EncodeBinary(w io.Writer) error {
	return nil
}

func (txn *Transaction) DecodeBinary(r io.Reader) error {
	return nil
}
