package server

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/network"
	"github.com/krehermann/goblockchain/protocol"
)

// convert an rpc that contains Contents of a Message to a decoded message.
func ExtractMessageFromRPC(rpc network.RPC) (*network.DecodedMessage, error) {
	// default handler makes strong assumptions about the format
	// of the data in the payload. this is fine and doesn't
	// effect extensibility -- another developer can
	// implement a rpc handler for whatever interpreteration of payload
	var out network.DecodedMessage

	msg, err := protocol.MessageFromRPC(rpc)
	if err != nil {
		return &out, err
	}
	if msg.Data == nil {
		return &out, fmt.Errorf("ExtractMessageFromRPC: nil data")
	}
	msgReader := bytes.NewReader(msg.Data)

	switch msg.Header {
	case protocol.MessageTypeTx:
		tx := new(core.Transaction)
		err := tx.Decode(core.NewGobTxDecoder(msgReader))
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode transacation from %s: %w", rpc.From, err)
		}
		out = network.DecodedMessage{
			From: rpc.From,
			Data: tx}

	case protocol.MessageTypeBlock:
		b := new(core.Block)
		err := b.Decode(core.NewDefaultBlockDecoder(msgReader))
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode block from %s: %w", rpc.From, err)
		}
		out = network.DecodedMessage{
			From: rpc.From,
			Data: b}
	case protocol.MessageTypeStatusRequest:
		sMsg := new(protocol.StatusMessageRequest)
		err := gob.NewDecoder(msgReader).Decode(sMsg)
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode status message from %s: %w", rpc.From, err)
		}

		out = network.DecodedMessage{
			From: rpc.From,
			Data: sMsg}

	case protocol.MessageTypeStatusResponse:
		sMsg := new(protocol.StatusMessageResponse)
		err := gob.NewDecoder(msgReader).Decode(sMsg)
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode status message from %s: %w", rpc.From, err)
		}
		out = network.DecodedMessage{
			From: rpc.From,
			Data: sMsg}

	case protocol.MessageTypeSubscribeRequest:
		sMsg := new(protocol.SubscribeMessageRequest)
		err := gob.NewDecoder(msgReader).Decode(sMsg)
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode subscribe request message from %s: %w", rpc.From, err)
		}

		out = network.DecodedMessage{
			From: rpc.From,
			Data: sMsg}

	case protocol.MessageTypeSubscribeResponse:
		sMsg := new(protocol.SubscribeMessageResponse)
		err := gob.NewDecoder(msgReader).Decode(sMsg)
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode subscribe response message from %s: %w", rpc.From, err)
		}
		out = network.DecodedMessage{
			From: rpc.From,
			Data: sMsg}

	case protocol.MessageTypeGetBlocksRequest:
		sMsg := new(protocol.GetBlocksRequest)
		err := gob.NewDecoder(msgReader).Decode(sMsg)
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode get block request message from %s: %w", rpc.From, err)
		}

		out = network.DecodedMessage{
			From: rpc.From,
			Data: sMsg}

	case protocol.MessageTypeGetBlocksResponse:
		sMsg := new(protocol.GetBlocksResponse)
		err := gob.NewDecoder(msgReader).Decode(sMsg)
		if err != nil {
			return &out, fmt.Errorf("HandleRPC: failed to decode get block response message from %s: %w", rpc.From, err)
		}
		out = network.DecodedMessage{
			From: rpc.From,
			Data: sMsg}

	default:
		return &out, fmt.Errorf("invalid message type %s", msg.Header)

	}
	return &out, nil
}
