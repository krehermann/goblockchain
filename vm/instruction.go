package vm

type Instruction byte

const (
	InstructionPush Instruction = 0x0a //10
	InstructionAdd  Instruction = 0x0b //11

)

// example
// 1 + 2 = 3
// 1
// push stack
// 1
// push stack
// add
// 3
