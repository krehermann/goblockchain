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

	b := make([]byte, len(msg))
	n, err := got.Payload.Read(b)
	assert.Equal(t, n, len(b))
	assert.NoError(t, err)
	assert.Equal(t, msg, b)
	assert.Equal(t, tra.addr, got.From)

}
