package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testList(t *testing.T) *List[int] {
	l := NewList[int]()
	for i := 0; i < 5; i += 1 {
		l.Append(i)
	}
	assert.Equal(t, 5, l.Len())
	return l
}
func TestList_Get(t *testing.T) {
	l := testList(t)

	for i := 0; i < l.Len(); i += 1 {
		got, err := l.Get(i)
		assert.NoError(t, err)
		assert.Equal(t, i, got)
	}
}

func TestList_Clear(t *testing.T) {
	l := testList(t)
	l.Clear()
	assert.Equal(t, 0, l.Len())
}

func TestList_Index(t *testing.T) {
	l := NewList[int]()
	for i := 0; i < 5; i += 1 {
		l.Append(i * 2)
	}
	assert.Equal(t, 5, l.Len())

	for i := 0; i < l.Len(); i += 1 {
		idx := l.Index(i * 2)
		assert.Equal(t, i, idx)
	}
}

func TestList_Delete(t *testing.T) {
	l := NewList[int]()
	for i := 0; i < 6; i += 1 {
		l.Append(i * 2)
	}
	assert.Equal(t, 6, l.Len())

	// delete in middle
	l.Delete(8)
	assert.Equal(t, 5, l.Len())
	got := make([]int, l.Len())
	want := []int{0, 2, 4, 6, 10}

	for i := 0; i < l.Len(); i += 1 {
		val, err := l.Get(i)
		assert.NoError(t, err)
		got[i] = val
	}
	assert.Equal(t, want, got)

	// delete last
	l.Delete(10)
	assert.Equal(t, 4, l.Len())
	got = make([]int, l.Len())
	want = []int{0, 2, 4, 6}

	for i := 0; i < l.Len(); i += 1 {
		val, err := l.Get(i)
		assert.NoError(t, err)
		got[i] = val
	}
	assert.Equal(t, want, got)

	// delete first
	l.Delete(0)
	assert.Equal(t, 3, l.Len())
	got = make([]int, l.Len())
	want = []int{2, 4, 6}

	for i := 0; i < l.Len(); i += 1 {
		val, err := l.Get(i)
		assert.NoError(t, err)
		got[i] = val
	}
	assert.Equal(t, want, got)

}
