package network

import (
	"fmt"
	"sync"

	"github.com/krehermann/goblockchain/types"
	"go.uber.org/zap"
)

type LocalTransport struct {
	addr types.NetAddr
	// transport responsible for maintaining and connecting to
	// peers, use a map to track them
	peers  map[types.NetAddr]*LocalTransport
	m      sync.RWMutex
	recvCh chan RPC
	logger *zap.Logger
}

func NewLocalTransport(addr types.NetAddr) *LocalTransport {
	return &LocalTransport{
		addr:   addr,
		recvCh: make(chan RPC, 1024),
		peers:  make(map[types.NetAddr]*LocalTransport),
		// TODO functional opts
		logger: zap.L().Named("transport").Named(string(addr)),
	}
}

func (lt *LocalTransport) Get(addr types.NetAddr) (Transport, bool) {
	lt.logger.Info("Get",
		zap.Any("want", addr),
		zap.Any("have", lt.peers))
	peer, exists := lt.peers[addr]
	return peer, exists
}

func (lt *LocalTransport) Recv() <-chan RPC {
	return lt.recvCh
}

func (lt *LocalTransport) Broadcast(payload Payload) error {
	for _, p := range lt.peers {
		if payload.data == nil {
			return fmt.Errorf("refusing to send nil payload")
		}
		err := lt.Send(p.Addr(), payload)
		if err != nil {
			return err
		}
	}
	return nil
}

// setup up bidirecional connection between transports
func (lt *LocalTransport) Connect(tr Transport) error {
	localTr, ok := tr.(*LocalTransport)
	if !ok {
		return fmt.Errorf("local transport can only connect to another local transport. got %v", tr)
	}

	lt.logger.Info("connect",
		zap.Any("me", lt.Addr()),
		zap.Any("i have", lt.peers),

		zap.Any("other", localTr.Addr()),
		zap.Any("other has", localTr.peers),
	)

	skip := func() bool {
		lt.m.RLock()
		defer lt.m.RUnlock()
		_, exists := lt.peers[tr.Addr()]
		return exists
	}
	if skip() {
		lt.logger.Named("localtransport").Info("skipping existing connection")

		return nil
	}
	lt.m.Lock()
	lt.peers[tr.Addr()] = localTr
	lt.m.Unlock()

	return tr.Connect(lt)

}

func (lt *LocalTransport) Addr() types.NetAddr {
	return lt.addr
}

func (lt *LocalTransport) Send(to types.NetAddr, payload Payload) error {
	lt.logger.Debug("send message",
		zap.String("from", string(lt.Addr())),
		zap.String("to", string(to)),
		zap.Int("msg length", len(payload.data)),
	)

	lt.m.RLock()
	defer lt.m.RUnlock()

	peer, exists := lt.peers[to]
	if !exists {
		return fmt.Errorf("%s unknown peer %s", lt.addr, to)
	}

	msg := RPC{
		From:    lt.addr,
		Content: payload.Reader(),
	}

	// this is kind of hacky b/c it's using the private
	// field rather than the interface.
	// the interface only support a recieve channel and can't be
	// written to
	peer.recvCh <- msg

	return nil
}
