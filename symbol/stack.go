package symbol

import (
	"debug/dwarf"
)

// Stores the subprogram information and its bounds.
type CallStackEntry struct {
	// Subprogram symbol
	Self *dwarf.Entry

	// Saved Stack Base Pointer
	Rbp uint64
}

// Stack storage
type CallStack struct {
	// Stores subprogram entries, last entry is the current scope
	Stack []*CallStackEntry
}

func NewCallStack() *CallStack {
	var cs = &CallStack{
		Stack: make([]*CallStackEntry, 0),
	}

	return cs
}

func (s *CallStack) Contains(Name string) bool {
	for _, entry := range s.Stack {
		if entry.Self.AttrField(dwarf.AttrName).Val == Name {
			return true
		}
	}
	return false
}

// Push Subprogram entry
func (s *CallStack) Push(Symbol *dwarf.Entry, Rbp uint64) {

	var NewEntry = &CallStackEntry{
		Self: Symbol,
		Rbp:  Rbp,
	}

	s.Stack = append(s.Stack, NewEntry)
}

// Pop subprogram entry
func (s *CallStack) Pop() {
	if len(s.Stack) == 0 {
		return
	}
	s.Stack = s.Stack[:len(s.Stack)-1]
}
