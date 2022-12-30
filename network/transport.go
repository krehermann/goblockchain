package network

type NetAddr string
type RPC struct {
	// message sent over transport layer
	From    NetAddr
	Payload []byte // this could an Any, but only sending bytes
}

// module in Transport
type Transport interface {
	Consume() <-chan RPC
	Connect(Transport) error
	SendMessage(NetAddr, []byte) error
	Addr() NetAddr
}
