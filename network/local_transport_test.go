package network

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {
	assert.Equal(t, 1, 1)

	tra := NewLocalTransport(LocalAddr("A"))
	trb := NewLocalTransport(LocalAddr("B"))

	err := tra.Connect(trb)
	assert.NoError(t, err)
	//assert.NoError(t, trb.Connect(tra))

	gotB, exists := tra.Get(trb.Addr())
	assert.True(t, exists)
	assert.Equal(t, trb, gotB)

	gotA, exists := trb.Get(tra.Addr())
	assert.True(t, exists)
	assert.Equal(t, tra, gotA)

	p := CreatePayload([]byte("hi"))
	assert.NoError(t, tra.Send(trb.addr, p))

	got := <-trb.Recv()

	b, err := ioutil.ReadAll(got.Content)
	assert.NoError(t, err)
	assert.Equal(t, p.data, b)
	assert.Equal(t, tra.addr, got.From)

}

func TestLocalTransport_Broadcast(t *testing.T) {

	trA := NewLocalTransport(LocalAddr("a"))
	trB := NewLocalTransport(LocalAddr("b"))
	trC := NewLocalTransport(LocalAddr("c"))

	trA.Connect(trB)
	trA.Connect(trC)
	payload := CreatePayload([]byte("hi there"))

	assert.NoError(t, trA.Broadcast(payload))

	rpcB := <-trB.Recv()
	assert.Equal(t, trA.Addr(), rpcB.From)

	gotB := make([]byte, len(payload.data))

	n, err := rpcB.Content.Read(gotB)
	assert.Equal(t, n, len(payload.data))

	assert.NoError(t, err)
	assert.Equal(t, payload.data, gotB)

	rpcC := <-trC.Recv()
	assert.Equal(t, trA.Addr(), rpcC.From)
	gotC, err := ioutil.ReadAll(rpcC.Content)
	assert.NoError(t, err)
	assert.Equal(t, payload.data, gotC)

}
