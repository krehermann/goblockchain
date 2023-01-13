package vm

import (
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStack(t *testing.T) {
	type args struct {
		opts []StackOpt
	}
	tests := []struct {
		name string
		args args
		want *Stack
	}{
		{
			name: "default",
			want: &Stack{
				ptr:   0,
				depth: 1024,
				data:  make([]any, 1024),
			},
		},
		{
			name: "depth opt",
			args: args{
				[]StackOpt{MaxStack(2)},
			},
			want: &Stack{
				ptr:   0,
				depth: 2,
				data:  make([]any, 2),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewStack(tt.args.opts...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewStack() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStack_Pop(t *testing.T) {
	type fields struct {
		lock  sync.RWMutex
		data  []any
		ptr   int
		depth int
	}
	tests := []struct {
		name      string
		fields    fields
		want      any
		wantErr   bool
		wantPanic bool
	}{

		{
			name:    "empty",
			wantErr: true,
		},
		{
			name: "start",
			fields: fields{
				data: []any{1, 7, 9},
				ptr:  1,
			},
			wantErr: false,
			want:    1,
		},
		{
			name: "middle",
			fields: fields{
				data: []any{2, 7, 9},
				ptr:  2,
			},
			wantErr: false,
			want:    7,
		},
		{
			name: "end",
			fields: fields{
				data: []any{2, 7, 9},
				ptr:  3,
			},
			wantErr: false,
			want:    9,
		},
		{
			name: "out of range",
			fields: fields{
				data: []any{2, 7, 9},
				ptr:  4,
			},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Stack{
				lock:  tt.fields.lock,
				data:  tt.fields.data,
				ptr:   tt.fields.ptr,
				depth: tt.fields.depth,
			}
			if tt.wantPanic {
				assert.Panics(t, func() { s.Pop() })
				return
			}
			got, err := s.Pop()
			if (err != nil) != tt.wantErr {
				t.Errorf("Stack.Pop() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Stack.Pop() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStackFunctional(t *testing.T) {
	max := 3
	vals := []int{0, 2, 4}
	s := NewStack(MaxStack(max))
	for i := 0; i < max; i += 1 {
		assert.NoError(t, s.Push(vals[i]))
		assert.Equal(t, i+1, s.Len())
	}

	// overflow
	assert.Error(t, s.Push("bad"))

	// pop all
	for i := 0; i < max; i += 1 {
		l := s.Len()
		assert.Equal(t, l, max-i)
		want := vals[l-1]
		got, err := s.Pop()
		assert.NoError(t, err)
		assert.Equal(t, got, want)
	}

	// underflow
	_, err := s.Pop()
	assert.Error(t, err)

	// reuse
	assert.NoError(t, s.Push("hi"))
	assert.Equal(t, 1, s.Len())
	// mixed type
	assert.NoError(t, s.Push([]byte("foo")))
	assert.Equal(t, 2, s.Len())

	byteVal, err := s.Pop()
	assert.NoError(t, err)
	assert.Equal(t, []byte("foo"), byteVal)

	strVal, err := s.Pop()
	assert.NoError(t, err)
	assert.Equal(t, "hi", strVal)

}
