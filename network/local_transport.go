package network

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

type LocalTransport struct {
	addr NetAddr
	// transport responsible for maintaining and connecting to
	// peers, use a map to track them
	peers     map[NetAddr]*LocalTransport
	m         sync.RWMutex
	consumeCh chan RPC
	logger    *zap.Logger
}

func NewLocalTransport(addr NetAddr) *LocalTransport {
	return &LocalTransport{
		addr:      addr,
		consumeCh: make(chan RPC, 1024),
		peers:     make(map[NetAddr]*LocalTransport),
		// TODO functional opts
		logger: zap.L().Named("transport").Named(string(addr)),
	}
}

func (lt *LocalTransport) Consume() <-chan RPC {
	return lt.consumeCh
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

// local transport can only connect to another local transport
// which is pretty obvious from the name. In this case,
// connect simply adds the input to the peer list
func (lt *LocalTransport) Connect(tr Transport) error {

	localTr, ok := tr.(*LocalTransport)
	if !ok {
		return fmt.Errorf("local transport can only connect to another local transport. got %v", tr)
	}

	setupErr := func() error {
		lt.m.Lock()
		defer lt.m.Unlock()

		lt.peers[tr.Addr()] = localTr
		return nil
	}()
	if setupErr != nil {
		return setupErr
	}
	return nil
}

func (lt *LocalTransport) Addr() NetAddr {
	return lt.addr
}

func (lt *LocalTransport) Send(to NetAddr, payload Payload) error {
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
	peer.consumeCh <- msg

	return nil
}
