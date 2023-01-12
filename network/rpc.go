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
	From NetAddr
	// this is totally generic
	Content io.Reader
}

// MessageType identities the type data enveloped in a Message
type MessageType byte

const (
	MessageTypeTx MessageType = iota
	MessageTypeBlock
)

var MessageTypes = []MessageType{MessageTypeTx, MessageTypeBlock}

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

// Message is default data representation via RPCs
type Message struct {
	// these need to be public so that there can be encoded/decoded
	Header MessageType
	Data   []byte
}

func NewMessage(t MessageType, data []byte) *Message {
	return &Message{
		Header: t,
		Data:   data,
	}
}

// gob encoding of the message struct.
func (m *Message) Bytes() ([]byte, error) {
	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(m)
	return buf.Bytes(), err
}

// generic deserialization for consuming RPCs
type DecodedMessage struct {
	From NetAddr
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

// convert an rpc that contains Contents of a Message to a decoded message.
func ExtractMessageFromRPC(rpc RPC) (*DecodedMessage, error) {
	// default handler makes strong assumptions about the format
	// of the data in the payload. this is fine and doesn't
	// effect extensibility -- another developer can
	// implement a rpc handler for whatever interpreteration of payload
	var out DecodedMessage

	msg, err := contentToMessage(rpc)
	if err != nil {
		return &out, err
	}
	msgReader := bytes.NewReader(msg.Data)

	switch msg.Header {
	case MessageTypeTx:
		tx := new(core.Transaction)
		err := tx.Decode(core.NewGobTxDecoder(msgReader))
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode transacation from %s: %w", rpc.From, err)
		}
		out = DecodedMessage{From: rpc.From, Data: tx}

	case MessageTypeBlock:
		b := new(core.Block)
		err := b.Decode(core.NewDefaultBlockDecoder(msgReader))
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode block from %s: %w", rpc.From, err)
		}
		out = DecodedMessage{From: rpc.From, Data: b}
	default:
		return &out, fmt.Errorf("invalid message type %s", msg.Header)

	}
	return &out, nil
}

func contentToMessage(rpc RPC) (Message, error) {
	msg := Message{}
	err := gob.NewDecoder(rpc.Content).Decode(&msg)
	if err != nil {
		return msg, fmt.Errorf("failed to extract message from rpc: %w", err)
	}
	return msg, nil
}

type RPCProcessor interface {
	ProcessMessage(*DecodedMessage) error
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

// helper to create a Message from block
// note to self: need to live in this package rather than func on tx
// because it is the right layering -- messages can wrap anything
// making a func on tx to create a message would be dependency invertion
func newMessageFromBlock(b *core.Block) (*Message, error) {
	buf := &bytes.Buffer{}
	// TODO does this encoder need to be a parameter?
	err := b.Encode(core.NewDefaultBlockEncoder(buf))
	if err != nil {
		return nil, err
	}
	return NewMessage(MessageTypeBlock, buf.Bytes()), nil
}
