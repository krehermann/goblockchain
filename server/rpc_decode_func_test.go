package server

import (
	"testing"
	"time"

	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/network"
	"github.com/krehermann/goblockchain/protocol"

	"github.com/stretchr/testify/assert"
)

func TestExtractMessageFromRPC(t *testing.T) {
	for _, tp := range protocol.MessageTypes {
		switch tp {
		case protocol.MessageTypeTx:
			testExtractTx(t)

		case protocol.MessageTypeBlock:
			testExtractBlock(t)
		case protocol.MessageTypeGetBlocksRequest:
			t.Logf("TODO %s", protocol.MessageTypeGetBlocksRequest)
		case protocol.MessageTypeGetBlocksResponse:
			t.Logf("TODO %s", protocol.MessageTypeGetBlocksResponse)
		case protocol.MessageTypeStatusRequest:
		case protocol.MessageTypeStatusResponse:
		case protocol.MessageTypeSubscribeRequest:
		case protocol.MessageTypeSubscribeResponse:
		default:
			assert.Failf(t, "unimplemented test for message type '%s'", tp.String())
		}
	}
}

func testExtractTx(t *testing.T) {
	tx := core.NewTransaction([]byte("tx"))
	msg, err := protocol.NewMessageFromTransaction(tx)
	assert.NoError(t, err)

	rpc := messageToRpc(t, msg)

	decodedMsg, err := ExtractMessageFromRPC(rpc)
	assert.NoError(t, err)

	got, ok := decodedMsg.Data.(*core.Transaction)
	assert.True(t, ok)
	assert.Equal(t, tx, got)
}

func testExtractBlock(t *testing.T) {
	b := core.NewBlock(
		&core.Header{
			Version:   1,
			Timestamp: uint64(time.Now().Unix()),
		},
		[]*core.Transaction{core.NewTransaction([]byte("junk"))},
	)

	msg, err := protocol.NewMessageFromBlock(b)
	assert.NoError(t, err)

	rpc := messageToRpc(t, msg)

	decodedMsg, err := ExtractMessageFromRPC(rpc)
	assert.NoError(t, err)

	got, ok := decodedMsg.Data.(*core.Block)
	assert.True(t, ok)
	assert.Equal(t, b, got)
}

func messageToRpc(t *testing.T, msg *protocol.Message) network.RPC {
	d, err := msg.Bytes()
	assert.NoError(t, err)
	payload := network.CreatePayload(d)

	return network.RPC{
		From:    network.LocalAddr("test-sender").String(),
		Content: payload.Reader(),
	}
}
