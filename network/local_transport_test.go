package network

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {
	assert.Equal(t, 1, 1)

	tra := NewLocalTransport("A")
	trb := NewLocalTransport("B")

	err := tra.Connect(trb)
	assert.NoError(t, err)
	//assert.NoError(t, trb.Connect(tra))

	assert.Equal(t, tra.peers[trb.addr], trb)
	assert.Equal(t, trb.peers[tra.addr], tra)

	p := CreatePayload([]byte("hi"))
	assert.NoError(t, tra.Send(trb.addr, p))

	got := <-trb.Consume()

	b, err := ioutil.ReadAll(got.Content)
	assert.NoError(t, err)
	assert.Equal(t, p.data, b)
	assert.Equal(t, tra.addr, got.From)

}

func TestLocalTransport_Broadcast(t *testing.T) {

	trA := NewLocalTransport("a")
	trB := NewLocalTransport("b")
	trC := NewLocalTransport("c")

	trA.Connect(trB)
	trA.Connect(trC)
	payload := CreatePayload([]byte("hi there"))

	assert.NoError(t, trA.Broadcast(payload))

	rpcB := <-trB.Consume()
	assert.Equal(t, trA.Addr(), rpcB.From)

	gotB := make([]byte, len(payload.data))

	n, err := rpcB.Content.Read(gotB)
	assert.Equal(t, n, len(payload.data))

	assert.NoError(t, err)
	assert.Equal(t, payload.data, gotB)

	rpcC := <-trC.Consume()
	assert.Equal(t, trA.Addr(), rpcC.From)
	gotC, err := ioutil.ReadAll(rpcC.Content)
	assert.NoError(t, err)
	assert.Equal(t, payload.data, gotC)

}
