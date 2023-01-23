package network

import (
	"bytes"
	"fmt"
	"io"
	"net"
)

// module in Transport
type Transport interface {
	Recv() <-chan RPC

	Connect(Transport) error
	Send(net.Addr, Payload) error
	Addr() net.Addr

	//IsConnected(net.Addr) bool
	Get(string) (Pipe, bool)
	Broadcast(Payload) error
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

type Pipe interface {
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	String() string
}

type pipeImpl struct {
	local  net.Addr
	remote net.Addr
}

func (p *pipeImpl) LocalAddr() net.Addr {
	return p.local
}

func (p *pipeImpl) RemoteAddr() net.Addr {
	return p.remote
}

func (p *pipeImpl) String() string {
	return fmt.Sprintf("%s (local) -> %s (remote)", p.local.String(), p.remote.String())
}
