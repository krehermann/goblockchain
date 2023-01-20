package network

import (
	"bytes"
	"io"
)

type NetAddr string

// module in Transport
type Transport interface {
	Recv() <-chan RPC

	Connect(Transport) error
	Send(NetAddr, Payload) error
	Addr() NetAddr

	Get(NetAddr) (Transport, bool)
	// transport is agnostic to the types in the payload
	Broadcast(payload Payload) error
}

type Payload struct {
	data []byte
}

func (p Payload) Reader() io.Reader {
	return bytes.NewReader(p.data)
}

func CreatePayload(d []byte) Payload {
	return Payload{data: d}
}
