package network

import (
	"bytes"
	"io"
	"net"
)

// module in Transport
type Transport interface {
	Recv() <-chan RPC

	Connect(Transport) error
	Send(net.Addr, Payload) error
	Addr() net.Addr

	IsConnected(net.Addr) bool
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
