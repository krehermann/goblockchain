package network

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/krehermann/goblockchain/core"
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

// helper to create a Message from block
// note to self: need to live in this package rather than func on tx
// because it is the right layering -- messages can wrap anything
// making a func on tx to create a message would be dependency invertion
func newMessageFromStatusMessageResponse(s *StatusMessageResponse) (*Message, error) {

	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(s)
	if err != nil {
		return nil, fmt.Errorf("failed to convert status message response to Message: %w", err)
	}
	return NewMessage(MessageTypeStatusResponse, buf.Bytes()), nil
}

func newMessageFromStatusMessageRequest(s *StatusMessageRequest) (*Message, error) {

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
func newMessageFromSubscribeMessageResponse(s *SubscribeMessageResponse) (*Message, error) {

	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(s)
	if err != nil {
		return nil, fmt.Errorf("failed to convert status message response to Message: %w", err)
	}
	return NewMessage(MessageTypeSubscribeResponse, buf.Bytes()), nil
}

func newMessageFromSubscribeMessageRequest(s *SubscribeMessageRequest) (*Message, error) {

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
func newMessageFromGetBlocksRequest(s *GetBlocksRequest) (*Message, error) {

	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(s)
	if err != nil {
		return nil, fmt.Errorf("failed to convert status message response to Message: %w", err)
	}
	return NewMessage(MessageTypeGetBlocksRequest, buf.Bytes()), nil
}

func newMessageFromGetBlocksResponse(s *GetBlocksResponse) (*Message, error) {

	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(s)
	if err != nil {
		return nil, fmt.Errorf("failed to convert status message request to Message: %w", err)
	}
	return NewMessage(MessageTypeGetBlocksResponse, buf.Bytes()), nil
}
