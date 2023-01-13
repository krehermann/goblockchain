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
					byte(InstructionPush),
					0x4,
					byte(InstructionPush),
					byte(InstructionAdd)},

				logger: zap.Must(zap.NewDevelopment()),
				stack:  NewStack(),
			},
			check: checkAdd,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := &VM{
				data:   tt.fields.data,
				ip:     tt.fields.ip,
				logger: tt.fields.logger,
				Stack:  tt.fields.stack,
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

func checkAdd(t *testing.T, vm *VM) {
	wantLen := 3
	got := vm.Stack.Len()
	assert.Equal(t, wantLen, got)

	addResult, err := vm.Stack.Read(vm.Stack.Len() - 1)
	assert.NoError(t, err)
	assert.Equal(t, 2+4, int(addResult))

}
