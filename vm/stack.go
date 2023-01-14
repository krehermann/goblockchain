package vm

import (
	"fmt"
	"sync"
)

type Stack struct {
	lock sync.RWMutex
	data []any
	ptr  int

	depth int
}

type StackOpt func(*Stack) *Stack

func MaxStack(max int) StackOpt {
	return func(s *Stack) *Stack {
		s.depth = max
		return s
	}
}

func NewStack(opts ...StackOpt) *Stack {
	s := &Stack{
		ptr:   0,
		depth: 1024,
	}
	for _, opt := range opts {
		s = opt(s)
	}
	s.data = make([]any, s.depth)
	return s
}

func (s *Stack) Push(b any) error {
	if s.ptr == s.depth {
		return fmt.Errorf("stack overflow")
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.data[s.ptr] = b
	s.ptr += 1

	return nil
}

func (s *Stack) Pop() (any, error) {
	if s.Empty() {
		return nil, fmt.Errorf("stack empty")
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	// ptr is at the next write slot, one ahead of the read slot
	b := s.data[s.ptr-1]
	s.ptr -= 1

	return b, nil
}

func (s *Stack) Empty() bool {
	s.lock.RLock()
	v := s.ptr
	s.lock.RUnlock()

	return v == 0
}

func (s *Stack) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ptr
}

func (s *Stack) Peek() (any, error) {
	return s.read(s.Len() - 1)
}

func (s *Stack) read(pos int) (any, error) {
	if pos >= s.Len() || pos < 0 {
		return nil, fmt.Errorf("read out of range len %d, pos %d", s.Len(), pos)
	}

	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.data[pos], nil

}
