package network

import (
	"testing"
	"time"

	"github.com/krehermann/goblockchain/core"
	"github.com/stretchr/testify/assert"
)

func TestExtractMessageFromRPC(t *testing.T) {
	for _, tp := range MessageTypes {
		switch tp {
		case MessageTypeTx:
			testExtractTx(t)

		case MessageTypeBlock:
			testExtractBlock(t)
		default:
			assert.Fail(t, "unimplemented test for message type '%s'", tp.String())
		}
	}
}

func testExtractTx(t *testing.T) {
	tx := core.NewTransaction([]byte("tx"))
	msg, err := newMessageFromTransaction(tx)
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

	msg, err := newMessageFromBlock(b)
	assert.NoError(t, err)

	rpc := messageToRpc(t, msg)

	decodedMsg, err := ExtractMessageFromRPC(rpc)
	assert.NoError(t, err)

	got, ok := decodedMsg.Data.(*core.Block)
	assert.True(t, ok)
	assert.Equal(t, b, got)
}

func messageToRpc(t *testing.T, msg *Message) RPC {
	d, err := msg.Bytes()
	assert.NoError(t, err)
	payload := CreatePayload(d)

	return RPC{
		From:    "test-sender",
		Content: payload.Reader(),
	}
}
