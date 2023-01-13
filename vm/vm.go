package vm

import (
	"fmt"

	"go.uber.org/zap"
)

type VM struct {
	// coming from transactions. probably instruction bytecode
	data []byte
	// instruction pointer
	ip int

	Stack  *Stack
	logger *zap.Logger
}

type VMOpt func(*VM) *VM

func LoggerOpt(l *zap.Logger) VMOpt {
	return func(vm *VM) *VM {
		vm.logger = l
		return vm
	}
}

func NewVM(data []byte, opts ...VMOpt) *VM {
	vm := &VM{
		data:   data,
		ip:     0,
		Stack:  NewStack(),
		logger: zap.L(),
	}

	for _, opt := range opts {
		vm = opt(vm)
	}

	vm.logger = vm.logger.Named("vm")

	return vm

}

func (vm *VM) Run() error {

	// loop over the data until all the instructions
	// are processed
	for {
		inst := Instruction(vm.data[vm.ip])

		vm.logger.Debug("instruction pointer",
			zap.Any("ip", inst))

		err := vm.Exec(inst)
		if err != nil {
			return fmt.Errorf("vm run: %w", err)
		}
		vm.ip++

		if vm.ip >= len(vm.data) {
			break
		}
	}
	return nil
}

func (vm *VM) Exec(inst Instruction) error {
	switch inst {
	case InstructionPush:
		d := vm.data[vm.ip-1]
		vm.logger.Debug("pushing to stack",
			zap.Any("b", d),
		)
		return vm.push(d)
	case InstructionAdd:
		vm.logger.Debug("add")

		a, err := vm.Stack.Read(vm.Stack.Len() - 1)
		if err != nil {
			return err
		}
		b, err := vm.Stack.Read(vm.Stack.Len() - 2)
		if err != nil {
			return err
		}

		aInt, bInt := int(a), int(b)
		val := aInt + bInt
		vm.logger.Debug("add",
			zap.Int("a", aInt),
			zap.Int("b", bInt),
			zap.Int("result", val),
		)

		return vm.Stack.Push(byte(val))
	}
	return nil
}

func (vm *VM) push(b byte) error {
	return vm.Stack.Push(b)
}
