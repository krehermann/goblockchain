package core

import (
	"fmt"
)

type Storager interface {
	Put(b *Block) error
	Get(h *Header) (*Block, error)
}

type MemStore struct {
	putChan  chan *Block
	readChan chan *getRequest
	data     map[*Header]*Block
}

type getRequest struct {
	key      *Header
	response chan<- *lookupResult
}

type lookupResult struct {
	b      *Block
	exists bool
}

func NewMemStore() *MemStore {
	s := &MemStore{
		putChan:  make(chan *Block),
		readChan: make(chan *getRequest),
		data:     make(map[*Header]*Block),
	}

	go s.handleAccess()
	return s
}

func (s *MemStore) handleAccess() {
	for {
		select {
		case b := <-s.putChan:
			s.data[b.Header] = b
		case req := <-s.readChan:
			b, ok := s.data[req.key]
			req.response <- &lookupResult{
				b:      b,
				exists: ok,
			}

		}
	}
}

func (ms *MemStore) Put(b *Block) error {

	ms.putChan <- b
	return nil
}

func (ms *MemStore) Get(h *Header) (*Block, error) {
	respCh := make(chan *lookupResult, 1)
	req := &getRequest{
		key:      h,
		response: respCh,
	}
	ms.readChan <- req
	resp := <-respCh
	if !resp.exists {
		return nil, fmt.Errorf("header %+v does not exist in store", h)
	}
	return resp.b, nil
}
