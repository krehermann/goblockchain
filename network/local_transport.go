package network

import (
	"fmt"
	"net"
	"sync"

	"go.uber.org/zap"
)

type LocalAddr string

func (l LocalAddr) Network() string { return "local" }
func (l LocalAddr) String() string  { return string(l) }

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

func (lt *LocalTransport) Get(id string) (Pipe, bool) {
	lt.m.RLock()
	tr, exists := lt.peers[LocalAddr(id)]
	lt.m.Unlock()
	if !exists {
		return nil, false
	}

	return &pipeImpl{
		local:  lt.addr,
		remote: tr.Addr()}, true
}
func (lt *LocalTransport) IsConnected(addr net.Addr) bool {
	lt.logger.Info("Get",
		zap.Any("want", addr),
		zap.Any("have", lt.peers))
	lt.m.RLock()
	defer lt.m.RUnlock()
	_, exists := lt.peers[addr.(LocalAddr)]
	return exists
}

func (lt *LocalTransport) Recv() <-chan RPC {
	return lt.recvCh
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

	if lt.IsConnected(tr.Addr()) {
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

func (lt *LocalTransport) Broadcast(payload Payload) error {
	lt.m.RLock()
	defer lt.m.RUnlock()

	for _, t := range lt.peers {
		err := lt.Send(t.Addr(), payload)
		if err != nil {
			lt.logger.Error("broadcast", zap.Error(err))
		}
	}
	return nil
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
		From:    lt.addr.String(),
		Content: payload.Reader(),
	}

	// this is kind of hacky b/c it's using the private
	// field rather than the interface.
	// the interface only support a recieve channel and can't be
	// written to
	peer.recvCh <- msg

	return nil
}
