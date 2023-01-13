package vm

import (
	"reflect"
	"testing"
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
				data:  make([]byte, 1024),
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
				data:  make([]byte, 2),
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
