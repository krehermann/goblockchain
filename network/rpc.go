package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/krehermann/goblockchain/core"
)

type RPC struct {
	// message sent over transport layer
	From    NetAddr
	Payload io.Reader
}

type MessageType byte

const (
	MessageTypeTx MessageType = iota
	MessageTypeBlock
)

func (mt MessageType) String() string {
	var out string
	switch mt {
	case MessageTypeBlock:
		out = "MessageTypeBlock"
	case MessageTypeTx:
		out = "MessageTypeTx"
	default:
		out = "unknown"
	}
	return out
}

type Message struct {
	Header MessageType
	Data   []byte
}

func NewMessage(t MessageType, data []byte) *Message {
	return &Message{
		Header: t,
		Data:   data,
	}
}

func (m *Message) Bytes() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(m)
	return buf.Bytes(), err
}

type RPCHandler interface {
	HandleRPC(rpc RPC) error
}

type DefaultRPCHandler struct {
	p RPCProcessor
}

func NewDefaultRPCHandler(p RPCProcessor) *DefaultRPCHandler {
	return &DefaultRPCHandler{
		p: p,
	}
}

func (h *DefaultRPCHandler) HandleRPC(rpc RPC) error {
	// default handler makes strong assumptions about the format
	// of the data in the payload. this is fine and doesn't
	// effect extensibility -- another developer can
	// implement a rpc handler for whatever interpreteration of payload

	msg := Message{}
	err := gob.NewDecoder(rpc.Payload).Decode(&msg)
	if err != nil {
		return fmt.Errorf("HandleRPC: failed to decode payload from %s: %w", rpc.From, err)
	}

	switch msg.Header {
	case MessageTypeTx:
		fmt.Println("handle tx")
		txReader := bytes.NewReader(msg.Data)
		tx := new(core.Transaction)
		err := tx.Decode(core.NewGobTxDecoder(txReader))
		if err != nil {
			return fmt.Errorf("HandleRPC: failed to decode transacation: %w", err)
		}
		h.p.ProcessTransaction(rpc.From, tx)
	default:
		return fmt.Errorf("invalid message type %s", msg.Header)

	}
	return nil
}

type RPCProcessor interface {
	ProcessTransaction(NetAddr, *core.Transaction) error
}
