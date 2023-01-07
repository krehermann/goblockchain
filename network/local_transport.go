package network

import (
	"bytes"
	"fmt"
	"sync"
)

type LocalTransport struct {
	addr NetAddr
	// transport responsible for maintaining and connecting to
	//peers, use a map to track them
	peers     map[NetAddr]*LocalTransport
	m         sync.RWMutex
	consumeCh chan RPC
}

func NewLocalTransport(addr NetAddr) *LocalTransport {
	return &LocalTransport{
		addr:      addr,
		consumeCh: make(chan RPC, 1024),
		peers:     make(map[NetAddr]*LocalTransport),
	}
}

func (lt *LocalTransport) Consume() <-chan RPC {
	return lt.consumeCh
}

func (lt *LocalTransport) Broadcast(payload []byte) error {
	for _, p := range lt.peers {
		err := lt.SendMessage(p.Addr(), payload)
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
	lt.m.Lock()
	defer lt.m.Unlock()

	localTr, ok := tr.(*LocalTransport)
	if !ok {
		return fmt.Errorf("local transport can only connect to another local transport. got %v", tr)
	}
	lt.peers[tr.Addr()] = localTr

	return nil
}

func (lt *LocalTransport) Addr() NetAddr {
	return lt.addr
}

func (lt *LocalTransport) SendMessage(to NetAddr, payload []byte) error {
	lt.m.RLock()
	defer lt.m.RUnlock()

	peer, exists := lt.peers[to]
	if !exists {
		return fmt.Errorf("%s unknown peer %s", lt.addr, to)
	}

	msg := RPC{
		From:    lt.addr,
		Payload: bytes.NewReader(payload),
	}

	// this is kind of hacky b/c it's using the private
	// field rather than the interface.
	// the interface only support a recieve channel and can't be
	// written to
	peer.consumeCh <- msg

	return nil
}
