package network

import (
	"fmt"
	"io/ioutil"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestTcpConnect(t *testing.T) {
	lg, err := zap.NewDevelopment()
	assert.NoError(t, err)
	zap.ReplaceGlobals(lg)

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	ad := l.Addr()
	t.Logf(" hack addr %s", l.Addr().String())
	l.Close()

	tra, err := NewTcpTransport(ad) //NewLocalTransport(LocalAddr("A"))
	require.NoError(t, err)

	l, err = net.Listen("tcp", ":0")
	require.NoError(t, err)
	ad2 := l.Addr()
	t.Logf(" hack addr %s", l.Addr().String())
	l.Close()

	trb, err := NewTcpTransport(ad2) //NewLocalTransport(LocalAddr("B"))
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		t.Logf("connecting a->b... %+v -> %+v ", tra.Addr(), trb.Addr())

		err = tra.Connect(trb)
		assert.NoError(t, err)
		t.Log("  done connecting a->b")

		assert.True(t, tra.IsConnected(trb.Addr()))

		p := CreatePayload([]byte(fmt.Sprintf("hi-%d", i)))
		assert.NoError(t, tra.Send(trb.addr, p))

		got := <-trb.Recv()

		bt, err := ioutil.ReadAll(got.Content)
		assert.NoError(t, err)
		assert.Equal(t, p.data, bt)
		assert.Equal(t, tra.addr.String(), got.From)
	}
}
