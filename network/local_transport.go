package network

import (
	"encoding/gob"
	"fmt"
	"net"
	"sync"

	"go.uber.org/zap"
)

type LocalAddr string

func (l LocalAddr) Network() string { return "local" }
func (l LocalAddr) String() string  { return string(l) }

// hack
func init() {
	var hackAddr LocalAddr
	gob.Register(hackAddr)
}

type LocalTransport struct {
	addr net.Addr
	// transport responsible for maintaining and connecting to
	// peers, use a map to track them
	peers  map[LocalAddr]*LocalTransport
	m      sync.RWMutex
	recvCh chan RPC
	logger *zap.Logger
}

func NewLocalTransport(addr net.Addr) *LocalTransport {
	return &LocalTransport{
		addr:   addr,
		recvCh: make(chan RPC, 1024),
		peers:  make(map[LocalAddr]*LocalTransport),
		// TODO functional opts
		logger: zap.L().Named("transport").Named(string(addr.String())),
	}
}

func (lt *LocalTransport) Get(addr net.Addr) (Transport, bool) {
	lt.logger.Info("Get",
		zap.Any("want", addr),
		zap.Any("have", lt.peers))
	peer, exists := lt.peers[addr.(LocalAddr)]
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
		_, exists := lt.peers[tr.Addr().(LocalAddr)]
		return exists
	}
	if skip() {
		lt.logger.Named("localtransport").Info("skipping existing connection")

		return nil
	}
	lt.m.Lock()
	lt.peers[tr.Addr().(LocalAddr)] = localTr
	lt.m.Unlock()

	// we need a directional connection, so the other local connection needs
	// to connect to us, too.
	return tr.Connect(lt)

}

func (lt *LocalTransport) Addr() net.Addr {
	return lt.addr
}

func (lt *LocalTransport) Send(to net.Addr, payload Payload) error {
	lt.logger.Debug("send message",
		zap.String("from", string(lt.addr.String())),
		zap.String("to", string(to.String())),
		zap.Int("msg length", len(payload.data)),
	)

	lt.m.RLock()
	defer lt.m.RUnlock()

	peer, exists := lt.peers[to.(LocalAddr)]
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
