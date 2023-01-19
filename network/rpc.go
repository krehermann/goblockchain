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
	if msg.Data == nil {
		return &out, fmt.Errorf("ExtractMessageFromRPC: nil data")
	}
	msgReader := bytes.NewReader(msg.Data)

	switch msg.Header {
	case MessageTypeTx:
		tx := new(core.Transaction)
		err := tx.Decode(core.NewGobTxDecoder(msgReader))
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode transacation from %s: %w", rpc.From, err)
		}
		out = DecodedMessage{
			From: rpc.From,
			Data: tx}

	case MessageTypeBlock:
		b := new(core.Block)
		err := b.Decode(core.NewDefaultBlockDecoder(msgReader))
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode block from %s: %w", rpc.From, err)
		}
		out = DecodedMessage{
			From: rpc.From,
			Data: b}
	case MessageTypeStatusRequest:
		sMsg := new(StatusMessageRequest)
		err := gob.NewDecoder(msgReader).Decode(sMsg)
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode status message from %s: %w", rpc.From, err)
		}

		out = DecodedMessage{
			From: rpc.From,
			Data: sMsg}

	case MessageTypeStatusResponse:
		sMsg := new(StatusMessageResponse)
		err := gob.NewDecoder(msgReader).Decode(sMsg)
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode status message from %s: %w", rpc.From, err)
		}
		out = DecodedMessage{
			From: rpc.From,
			Data: sMsg}

	case MessageTypeSubscribeRequest:
		sMsg := new(SubscribeMessageRequest)
		err := gob.NewDecoder(msgReader).Decode(sMsg)
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode subscribe request message from %s: %w", rpc.From, err)
		}

		out = DecodedMessage{
			From: rpc.From,
			Data: sMsg}

	case MessageTypeSubscribeResponse:
		sMsg := new(SubscribeMessageResponse)
		err := gob.NewDecoder(msgReader).Decode(sMsg)
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode subscribe response message from %s: %w", rpc.From, err)
		}
		out = DecodedMessage{
			From: rpc.From,
			Data: sMsg}

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
