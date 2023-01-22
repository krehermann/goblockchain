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

	assert.True(t, tra.IsConnected(trb.Addr()))

	assert.True(t, trb.IsConnected(tra.Addr()))

	p := CreatePayload([]byte("hi"))
	assert.NoError(t, tra.Send(trb.addr, p))

	got := <-trb.Recv()

	b, err := ioutil.ReadAll(got.Content)
	assert.NoError(t, err)
	assert.Equal(t, p.data, b)
	assert.Equal(t, tra.addr.String(), got.From)

}
