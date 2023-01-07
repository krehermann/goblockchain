package network

type NetAddr string

// module in Transport
type Transport interface {
	Consume() <-chan RPC
	Connect(Transport) error
	SendMessage(NetAddr, []byte) error
	Addr() NetAddr
	// transport is agnostic to the types in the payload
	Broadcast(payload []byte) error
}
