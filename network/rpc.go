package network

import (
	"bytes"
	"encoding/gob"
	"io"
)

type RPC struct {
	// message sent over transport layer
	//	From net.Addr
	From string
	// this is totally generic
	Content io.Reader
}

// generic deserialization for consuming RPCs
type DecodedMessage struct {
	From string
	// this is generic because the transport layer can send anything
	Data any
}

// Bytes is the gob-encoding of the Data field
func (d *DecodedMessage) Bytes() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(d.Data)
	return buf.Bytes(), err
}

type RPCDecodeFunc func(RPC) (*DecodedMessage, error)

type RPCProcessor interface {
	ProcessMessage(*DecodedMessage) error
}
