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

// helper to create a Message from Tx
// note to self: need to live in this package rather than func on tx
// because it is the right layering -- messages can wrap anything
// making a func on tx to create a message would be dependency invertion
func newMessageFromTransaction(tx *core.Transaction) (*Message, error) {
	buf := &bytes.Buffer{}
	// TODO does this encoder need to be a parameter?
	err := tx.Encode(core.NewGobTxEncoder(buf))
	if err != nil {
		return nil, err
	}
	return NewMessage(MessageTypeTx, buf.Bytes()), nil

}

type DecodedMessage struct {
	From NetAddr
	Data any
}

func (d *DecodedMessage) Bytes() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(d.Data)
	return buf.Bytes(), err
}

type RPCDecodeFunc func(RPC) (*DecodedMessage, error)

func DefaultRPCDecodeFunc(rpc RPC) (*DecodedMessage, error) {
	// default handler makes strong assumptions about the format
	// of the data in the payload. this is fine and doesn't
	// effect extensibility -- another developer can
	// implement a rpc handler for whatever interpreteration of payload

	msg := Message{}
	err := gob.NewDecoder(rpc.Payload).Decode(&msg)
	var out DecodedMessage
	if err != nil {
		return &out, fmt.Errorf("HandleRPC: failed to decode payload from %s: %w", rpc.From, err)
	}

	switch msg.Header {
	case MessageTypeTx:
		txReader := bytes.NewReader(msg.Data)
		tx := new(core.Transaction)
		err := tx.Decode(core.NewGobTxDecoder(txReader))
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode transacation from %s: %w", rpc.From, err)
		}
		out = DecodedMessage{From: rpc.From, Data: tx}
	default:
		return &out, fmt.Errorf("invalid message type %s", msg.Header)

	}
	return &out, nil
}

type RPCProcessor interface {
	ProcessMessage(*DecodedMessage) error
}
