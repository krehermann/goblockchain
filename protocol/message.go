package protocol

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/network"
)

// MessageType identities the type data enveloped in a Message
type MessageType byte

const (
	MessageTypeTx MessageType = iota
	MessageTypeBlock
	// status is message for onboarding to the network
	MessageTypeStatusRequest
	MessageTypeStatusResponse

	MessageTypeSubscribeResponse
	MessageTypeSubscribeRequest

	MessageTypeGetBlocksRequest
	MessageTypeGetBlocksResponse
)

var MessageTypes = []MessageType{
	MessageTypeTx,
	MessageTypeBlock,
	MessageTypeStatusRequest,
	MessageTypeStatusResponse,
	MessageTypeSubscribeRequest,
	MessageTypeSubscribeResponse,

	MessageTypeGetBlocksRequest,
	MessageTypeGetBlocksResponse,
}

func (mt MessageType) String() string {
	var out string
	switch mt {
	case MessageTypeBlock:
		out = "MessageTypeBlock"
	case MessageTypeTx:
		out = "MessageTypeTx"
	case MessageTypeStatusRequest:
		out = "MessageTypeStatusRequest"
	case MessageTypeStatusResponse:
		out = "MessageTypeStatusResponse"
	case MessageTypeSubscribeRequest:
		out = "MessageTypeSubscribeRequest"
	case MessageTypeSubscribeResponse:
		out = "MessageTypeSubscribeResponse"
	case MessageTypeGetBlocksRequest:
		out = "MessageTypeGetBlocksRequest"
	case MessageTypeGetBlocksResponse:
		out = "MessageTypeGetBlocksResponse"
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

// helper to create a Message from Tx
// note to self: need to live in this package rather than func on tx
// because it is the right layering -- messages can wrap anything
// making a func on tx to create a message would be dependency invertion
func NewMessageFromTransaction(tx *core.Transaction) (*Message, error) {
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
func NewMessageFromBlock(b *core.Block) (*Message, error) {
	buf := &bytes.Buffer{}
	// TODO does this encoder need to be a parameter?
	err := b.Encode(core.NewDefaultBlockEncoder(buf))
	if err != nil {
		return nil, err
	}
	return NewMessage(MessageTypeBlock, buf.Bytes()), nil
}

// helper to create a Message from block
// note to self: need to live in this package rather than func on tx
// because it is the right layering -- messages can wrap anything
// making a func on tx to create a message would be dependency invertion
func NewMessageFromStatusMessageResponse(s *StatusMessageResponse) (*Message, error) {

	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(s)
	if err != nil {
		return nil, fmt.Errorf("failed to convert status message response to Message: %w", err)
	}
	return NewMessage(MessageTypeStatusResponse, buf.Bytes()), nil
}

func NewMessageFromStatusMessageRequest(s *StatusMessageRequest) (*Message, error) {

	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(s)
	if err != nil {
		return nil, fmt.Errorf("failed to convert status message request to Message: %w", err)
	}
	return NewMessage(MessageTypeStatusRequest, buf.Bytes()), nil
}

// helper to create a Message from block
// note to self: need to live in this package rather than func on tx
// because it is the right layering -- messages can wrap anything
// making a func on tx to create a message would be dependency invertion
func NewMessageFromSubscribeMessageResponse(s *SubscribeMessageResponse) (*Message, error) {

	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(s)
	if err != nil {
		return nil, fmt.Errorf("failed to convert status message response to Message: %w", err)
	}
	return NewMessage(MessageTypeSubscribeResponse, buf.Bytes()), nil
}

func NewMessageFromSubscribeMessageRequest(s *SubscribeMessageRequest) (*Message, error) {

	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(s)
	if err != nil {
		return nil, fmt.Errorf("failed to convert status message request to Message: %w", err)
	}
	return NewMessage(MessageTypeSubscribeRequest, buf.Bytes()), nil
}

// helper to create a Message from block
// note to self: need to live in this package rather than func on tx
// because it is the right layering -- messages can wrap anything
// making a func on tx to create a message would be dependency invertion
func NewMessageFromGetBlocksRequest(s *GetBlocksRequest) (*Message, error) {

	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(s)
	if err != nil {
		return nil, fmt.Errorf("failed to convert status message response to Message: %w", err)
	}
	return NewMessage(MessageTypeGetBlocksRequest, buf.Bytes()), nil
}

func NewMessageFromGetBlocksResponse(s *GetBlocksResponse) (*Message, error) {

	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(s)
	if err != nil {
		return nil, fmt.Errorf("failed to convert status message request to Message: %w", err)
	}
	return NewMessage(MessageTypeGetBlocksResponse, buf.Bytes()), nil
}

func MessageFromRPC(rpc network.RPC) (Message, error) {
	msg := Message{}
	err := gob.NewDecoder(rpc.Content).Decode(&msg)
	if err != nil {
		return msg, fmt.Errorf("failed to extract message from rpc: %w", err)
	}
	return msg, nil
}
