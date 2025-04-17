package main

import (
	"debug/dwarf"
	"syscall"
)

// Stack storage
type CallStack struct {
	// Stores subprogram entries
	Stack []*dwarf.Entry
	// Stores subprogram aligments to calculate symbol addresses
	FrameOffset []int
}

func NewCallStack() *CallStack {
	var st []*dwarf.Entry
	var fb []int
	var cs = &CallStack{
		Stack:       st,
		FrameOffset: fb,
	}

	return cs
}

func (s *CallStack) Contains(Name string) bool {
	for _, entry := range s.Stack {
		if entry.AttrField(dwarf.AttrName).Val == Name {
			return true
		}
	}
	return false
}

// Push Subprogram entry and its aligment
// If aligment is same as before, enter 0.
func (s *CallStack) Push(Symbol *dwarf.Entry, CFAoffset int) {
	s.Stack = append(s.Stack, Symbol)
	s.FrameOffset = append(s.FrameOffset, CFAoffset)
}

// Pop subprogram entry and aligment from the top.
func (s *CallStack) Pop() {
	if len(s.Stack) == 0 {
		return
	}
	s.Stack = s.Stack[:len(s.Stack)-1]
	s.FrameOffset = s.FrameOffset[:len(s.FrameOffset)-1]
}

// When data alignment is needed for a CFA, it happens int the first three rows of the assembly code.
// eg.:
//
//			push RBP -> becomes INT3 at entry
//			mov RBP, RSP
//	     	sub	RSP, 0x10
//
// The offset inside the canonical frame address is the value subtracted from RSP.
// INT3 will overwrite the push RBP instruction, after replacing we stop over.
// If RSPs value changes in the next step, return the difference, or 0 if nothing changes
// Call StepOverBreakpoint before calling this function!
func GetCFAOffset(pid int, registers *syscall.PtraceRegs) int {
	// Execute mov RBP, RSP
	syscall.PtraceSingleStep(pid)

	syscall.PtraceGetRegs(pid, registers)

	// Get RSP before potentially modified
	prev_rsp := registers.Rsp

	// Execute last instruction where RSP might change
	syscall.PtraceSingleStep(pid)

	syscall.PtraceGetRegs(pid, registers)

	return (int)(prev_rsp - registers.Rsp)
}
