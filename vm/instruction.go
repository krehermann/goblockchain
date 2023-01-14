package vm

type Instruction byte

const (
	InstructionPushInt   Instruction = 0x0a //10
	InstructionAddInt    Instruction = 0x0b //11
	InstructionPushBytes Instruction = 0x0c
	InstructionPack      Instruction = 0x0d
	InstructionStore     Instruction = 0x0f
	InstructionGet       Instruction = 0x10
)

// example
// 1 + 2 = 3
// 1
// push stack
// 1
// push stack
// add
// 3
