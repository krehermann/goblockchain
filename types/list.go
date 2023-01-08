package types

import (
	"fmt"
	"reflect"
)

type List[T any] struct {
	data []T
}

func NewList[T any]() *List[T] {
	return &List[T]{
		data: make([]T, 0),
	}
}

func (l *List[T]) Get(idx int) (T, error) {
	var empty T
	if idx > len(l.data)-1 || idx < 0 {
		return empty, fmt.Errorf("index out of range. idx %d len %d",
			idx,
			len(l.data))
	}

	return l.data[idx], nil
}

func (l *List[T]) Append(val T) {
	l.data = append(l.data, val)
}

func (l *List[T]) Clear() {
	l.data = []T{}
}

// return first index of val, or -1
func (l *List[T]) IndexOf(val T) int {
	for idx, elem := range l.data {
		if reflect.DeepEqual(elem, val) {
			return idx
		}
	}
	return -1
}

func (l *List[T]) Delete(val T) {
	idx := l.IndexOf(val)
	if idx != -1 {
		l.DeleteAt(idx)
	}

}

func (l *List[T]) DeleteAt(idx int) {
	// make out of range idx innocous
	if idx < 0 || idx >= len(l.data) {
		return
	}

	out := l.data[:idx]
	// append remainer if idx is not last element
	if idx != len(l.data)-1 {
		out = append(out, l.data[idx+1:]...)
	}
	l.data = out
}

func (l *List[T]) Contains(val T) bool {
	for _, elem := range l.data {
		if reflect.DeepEqual(elem, val) {
			return true
		}
	}
	return false
}

func (l *List[T]) Last() T {
	return l.data[len(l.data)-1]
}

func (l *List[T]) Len() int {
	return len(l.data)
}
