package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {
	assert.Equal(t, 1, 1)

	tra := NewLocalTransport("A")
	trb := NewLocalTransport("B")

	err := tra.Connect(trb)
	assert.NoError(t, err)
	assert.NoError(t, trb.Connect(tra))

	assert.Equal(t, tra.peers[trb.addr], trb)
	assert.Equal(t, trb.peers[tra.addr], tra)

	msg := []byte("hi")
	assert.NoError(t, tra.SendMessage(trb.addr, msg))

	got := <-trb.Consume()

	assert.Equal(t, msg, got.Payload)
	assert.Equal(t, tra.addr, got.From)

}
