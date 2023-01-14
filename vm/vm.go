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

	Stack *Stack

	contractState *State
	logger        *zap.Logger
}

type VMOpt func(*VM) *VM

func LoggerOpt(l *zap.Logger) VMOpt {
	return func(vm *VM) *VM {
		vm.logger = l
		return vm
	}
}

func ContractStateOpt(cs *State) VMOpt {
	return func(vm *VM) *VM {
		vm.contractState = cs
		return vm
	}
}

func NewVM(data []byte, opts ...VMOpt) *VM {
	vm := &VM{
		data:          data,
		ip:            0,
		Stack:         NewStack(),
		logger:        zap.L(),
		contractState: NewState(),
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
	case InstructionPushInt:
		d := int(vm.data[vm.ip-1])
		vm.logger.Debug("pushing int to stack",
			zap.Int("d", d),
		)
		return vm.Stack.Push(d)
	case InstructionAddInt:
		vm.logger.Debug("add")

		a, err := vm.Stack.Pop()
		if err != nil {
			return err
		}
		b, err := vm.Stack.Pop()
		if err != nil {
			return err
		}
		// TODO this is needs to be fixed to handle broader type sytem
		// after the stack can hold []any
		// current there is skew between the []data byte in the tx
		// and the []data any in the stack. probably need
		// as deserializer for the tx data...
		aInt, bInt := a.(int), b.(int)
		val := aInt + bInt
		vm.logger.Debug("add",
			zap.Int("a", int(aInt)),
			zap.Int("b", int(bInt)),
			zap.Int("result", int(val)),
		)

		return vm.Stack.Push(val)
	case InstructionPushBytes:
		d := byte(vm.data[vm.ip-1])
		vm.logger.Debug("pushing byte to stack",
			zap.Any("d", d),
		)
		return vm.Stack.Push(d)

	case InstructionPack:
		// need to get byte length and read them
		v, err := vm.Stack.Pop()
		if err != nil {
			return err
		}

		l := v.(int)
		data := make([]byte, l)
		//		stackPosStart := vm.Stack.Len() - 2
		for i := 0; i < l; i++ {

			// insert in reverse order
			val, err := vm.Stack.Pop() // vm.Stack.Read(stackPosStart - i)
			if err != nil {
				return err
			}
			data[l-(i+1)] = val.(byte)
		}
		vm.Stack.Push(data)
	case InstructionStore:

		val, err := vm.Stack.Pop()
		if err != nil {
			return err
		}
		key, err := vm.Stack.Pop()
		if err != nil {
			return err
		}
		k := mustBeKey(key)
		vm.logger.Debug("store",
			zap.Any("k", k),
			zap.Any("v", val),
		)
		//		return vm.contractState.Put(mustBeKey(key), mustBeVal(val))
		return vm.contractState.Put(k, val)

	}
	return nil
}

func mustBeKey(k any) Key {
	s := string(k.([]byte))
	return Key(s)
}

/*
// hack. need way to convert data from the stack to
// something that can be put into State
func mustBeVal(v any) []byte {
	switch t := v.(type) {
	case []byte:
		return t
	case int:
		return []byte{byte(t)}
	}
	return nil
}
*/
