package vm

import (
	"fmt"
	"sync"
)

type Key string

type State struct {
	lock sync.RWMutex
	// we are using any b/c it's easier with
	// our stack implementation. we may need
	// to use
	//	data map[Key][]byte
	// which will require serialization for all supported types
	// when taking from the stack and putting into the store

	data map[Key]any
}

func NewState() *State {
	return &State{
		data: make(map[Key]any),
		lock: sync.RWMutex{},
	}
}

func (s *State) Put(k Key, v any) error {
	//	s.lock.Lock()
	//	defer s.lock.Unlock()
	s.data[k] = v
	return nil
}

func (s *State) Delete(k Key) error {
	delete(s.data, k)
	return nil
}

func (s *State) Get(k Key) (any, error) {
	val, exists := s.data[k]
	if !exists {
		return []byte{}, fmt.Errorf("key '%s' does not exist", k)
	}
	return val, nil
}
