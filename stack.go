package main

import (
	"debug/dwarf"
)

// Stack storage
type CallStack struct {
	// Stores subprogram entries
	Stack []*dwarf.Entry
}

func NewCallStack() *CallStack {
	var st []*dwarf.Entry
	var cs = &CallStack{
		Stack: st,
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
func (s *CallStack) Push(Symbol *dwarf.Entry) {
	s.Stack = append(s.Stack, Symbol)
}

// Pop subprogram entry and aligment from the top.
func (s *CallStack) Pop() {
	if len(s.Stack) == 0 {
		return
	}
	s.Stack = s.Stack[:len(s.Stack)-1]
}
