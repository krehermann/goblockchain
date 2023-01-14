package vm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestVM_Run(t *testing.T) {
	type fields struct {
		data   []byte
		ip     int
		logger *zap.Logger
		stack  *Stack
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		check   func(*testing.T, *VM)
	}{
		{
			name: "add 2 + 4",
			fields: fields{
				data: []byte{
					0x2,
					byte(InstructionPushInt),
					0x4,
					byte(InstructionPushInt),
					byte(InstructionAddInt)},

				logger: zap.Must(zap.NewDevelopment()),
				stack:  NewStack(),
			},
			check: checkAdd2_4,
		},

		{
			name: "add 0 + 1",
			fields: fields{
				data: []byte{
					0x0,
					byte(InstructionPushInt),
					0x1,
					byte(InstructionPushInt),
					byte(InstructionAddInt)},

				logger: zap.Must(zap.NewDevelopment()),
				stack:  NewStack(),
			},
			check: checkAdd0_1,
		},

		{
			name: "one byte",
			fields: fields{
				data: []byte{
					byte('a'),
					byte(InstructionPushBytes),
				},

				logger: zap.Must(zap.NewDevelopment()),
				stack:  NewStack(),
			},
			check: checkPushOneByte,
		},

		{
			name: "pack",
			fields: fields{
				data: []byte{
					byte('a'),
					byte(InstructionPushBytes),
					byte('b'),
					byte(InstructionPushBytes),
					byte(2),
					byte(InstructionPushInt),
					byte(InstructionPack),
				},

				logger: zap.Must(zap.NewDevelopment()),
				stack:  NewStack(),
			},
			check: checkPack,
		},

		{
			name: "store xx:7",
			fields: fields{
				data: []byte{
					// make the xx key
					byte('x'),
					byte(InstructionPushBytes),
					byte('x'),
					byte(InstructionPushBytes),
					byte(2),
					byte(InstructionPushInt),
					byte(InstructionPack),
					// make the val
					byte(7),
					byte(InstructionPushInt),
					// store
					byte(InstructionStore),
				},

				logger: zap.Must(zap.NewDevelopment()),
				stack:  NewStack(),
			},
			check: checkStorexx7,
		},

		{
			name: "store mutliple xx:7 yy:3",
			fields: fields{
				data: []byte{
					// make the xx key
					byte('x'),
					byte(InstructionPushBytes),
					byte('x'),
					byte(InstructionPushBytes),
					byte(2),
					byte(InstructionPushInt),
					byte(InstructionPack),
					// make the val
					byte(7),
					byte(InstructionPushInt),
					// store
					byte(InstructionStore),

					// make the xx key
					byte('y'),
					byte(InstructionPushBytes),
					byte('y'),
					byte(InstructionPushBytes),
					byte(2),
					byte(InstructionPushInt),
					byte(InstructionPack),
					// make the val
					byte(3),
					byte(InstructionPushInt),
					// store
					byte(InstructionStore),
				},

				logger: zap.Must(zap.NewDevelopment()),
				stack:  NewStack(),
			},
			check: checkStoreMultiple,
		},

		{
			name: "get",
			fields: fields{
				data: []byte{
					// make the x key
					byte('x'),
					byte(InstructionPushBytes),
					byte(1),
					byte(InstructionPushInt),
					byte(InstructionPack),
					// make the val
					byte(7),
					byte(InstructionPushInt),
					// store
					byte(InstructionStore),

					byte('x'),
					byte(InstructionPushBytes),
					byte(1),
					byte(InstructionPushInt),
					byte(InstructionPack),
					byte(InstructionGet),
				},

				logger: zap.Must(zap.NewDevelopment()),
				stack:  NewStack(),
			},
			check: checkGet,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := &VM{
				data:          tt.fields.data,
				ip:            tt.fields.ip,
				logger:        tt.fields.logger,
				Stack:         tt.fields.stack,
				contractState: NewState(),
			}
			if err := vm.Run(); (err != nil) != tt.wantErr {
				t.Errorf("VM.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.check != nil {
				tt.check(t, vm)
			}
		})
	}
}

func checkAdd2_4(t *testing.T, vm *VM) {
	wantLen := 1
	got := vm.Stack.Len()
	assert.Equal(t, wantLen, got)

	addResult, err := vm.Stack.Pop()
	assert.NoError(t, err)
	assert.Equal(t, 2+4, addResult)

}

func checkAdd0_1(t *testing.T, vm *VM) {
	wantLen := 1
	got := vm.Stack.Len()
	assert.Equal(t, wantLen, got)

	addResult, err := vm.Stack.Pop()
	assert.NoError(t, err)
	assert.Equal(t, 0+1, addResult)

}

func checkPushOneByte(t *testing.T, vm *VM) {
	wantLen := 1
	got := vm.Stack.Len()
	assert.Equal(t, wantLen, got)
	val, err := vm.Stack.Pop()
	assert.NoError(t, err)
	assert.Equal(t, byte('a'), val)

}

func checkPack(t *testing.T, vm *VM) {
	wantLen := 1
	got := vm.Stack.Len()
	assert.Equal(t, wantLen, got)
	val, err := vm.Stack.Pop()
	assert.NoError(t, err)
	assert.Equal(t, []byte{'a', 'b'}, val)
}

func checkStorexx7(t *testing.T, vm *VM) {

	want := NewState()
	assert.NoError(t, want.Put(Key("xx"), 7))

	got := vm.contractState

	assert.Equal(t, want.data, got.data)

}

func checkStoreMultiple(t *testing.T, vm *VM) {

	want := NewState()
	assert.NoError(t, want.Put(Key("xx"), 7))
	assert.NoError(t, want.Put(Key("yy"), 3))

	got := vm.contractState
	assert.Equal(t, want.data, got.data)

}

// helper that checks the stack after x:7 is stored and x is Get'd
func checkGet(t *testing.T, vm *VM) {
	want := NewState()
	expected := 7
	assert.NoError(t, want.Put(Key("x"), expected))

	got := vm.contractState
	assert.Equal(t, want.data, got.data)

	val, err := vm.Stack.Pop()
	assert.NoError(t, err)
	assert.Equal(t, expected, val)
}
