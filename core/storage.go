package core

import (
	"fmt"
)

type GenericStorager[T comparable, V any] interface {
	Put(T, V) error
	Get(T) (V, error)
}

type genericPut[T comparable, V any] struct {
	key T
	val V
}

type GenericMemStore[T comparable, V any] struct {
	putChan  chan *genericPut[T, V]
	readChan chan *genericGetRequest[T, V]
	data     map[T]V
}

type genericGetRequest[T comparable, V any] struct {
	key      T
	response chan<- *genericLookupResult[V]
}

type genericLookupResult[V any] struct {
	val    any
	exists bool
}

func NewGenericMemStore[T comparable, V any]() *GenericMemStore[T, V] {
	s := &GenericMemStore[T, V]{
		putChan:  make(chan *genericPut[T, V]),
		readChan: make(chan *genericGetRequest[T, V]),
		data:     make(map[T]V),
	}

	go s.handleAccess()
	return s
}

func (s *GenericMemStore[T, V]) handleAccess() {
	for {
		select {
		case p := <-s.putChan:
			s.data[p.key] = p.val
		case req := <-s.readChan:
			val, ok := s.data[req.key]
			req.response <- &genericLookupResult[V]{
				val:    val,
				exists: ok,
			}

		}
	}
}

func (ms *GenericMemStore[T, V]) Put(key T, val V) error {

	ms.putChan <- &genericPut[T, V]{
		key: key,
		val: val,
	}
	return nil
}

func (ms *GenericMemStore[T, V]) Get(key T) (V, error) {
	respCh := make(chan *genericLookupResult[V], 1)
	req := &genericGetRequest[T, V]{
		key:      key,
		response: respCh,
	}
	ms.readChan <- req
	resp := <-respCh
	var deflt V
	if !resp.exists {
		return deflt, fmt.Errorf("key %+v does not exist in store", key)
	}
	return resp.val.(V), nil
}
