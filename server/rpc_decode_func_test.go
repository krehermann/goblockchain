package server

import (
	"testing"
	"time"

	"github.com/krehermann/goblockchain/api"
	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/network"

	"github.com/stretchr/testify/assert"
)

func TestExtractMessageFromRPC(t *testing.T) {
	for _, tp := range api.MessageTypes {
		switch tp {
		case api.MessageTypeTx:
			testExtractTx(t)

		case api.MessageTypeBlock:
			testExtractBlock(t)
		case api.MessageTypeGetBlocksRequest:
			t.Logf("TODO %s", api.MessageTypeGetBlocksRequest)
		case api.MessageTypeGetBlocksResponse:
			t.Logf("TODO %s", api.MessageTypeGetBlocksResponse)
		case api.MessageTypeStatusRequest:
		case api.MessageTypeStatusResponse:
		case api.MessageTypeSubscribeRequest:
		case api.MessageTypeSubscribeResponse:
		default:
			assert.Failf(t, "unimplemented test for message type '%s'", tp.String())
		}
	}
}

func testExtractTx(t *testing.T) {
	tx := core.NewTransaction([]byte("tx"))
	msg, err := api.NewMessageFromTransaction(tx)
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

	msg, err := api.NewMessageFromBlock(b)
	assert.NoError(t, err)

	rpc := messageToRpc(t, msg)

	decodedMsg, err := ExtractMessageFromRPC(rpc)
	assert.NoError(t, err)

	got, ok := decodedMsg.Data.(*core.Block)
	assert.True(t, ok)
	assert.Equal(t, b, got)
}

func messageToRpc(t *testing.T, msg *api.Message) network.RPC {
	d, err := msg.Bytes()
	assert.NoError(t, err)
	payload := network.CreatePayload(d)

	return network.RPC{
		From:    "test-sender",
		Content: payload.Reader(),
	}
}
