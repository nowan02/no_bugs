package symbol

import (
	"debug/dwarf"
	"encoding/binary"
)

// Stores the subprogram information and its base from where local vars can be calculated.
type CallStackEntry struct {
	// Subprogram symbol
	Self *dwarf.Entry

	// Saved Stack Base Pointer
	ReturnAddress uint64
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

// Check if an entry with such a base address is already in the stack.
func (s *CallStack) ContainsAddress(Address uint64) bool {
	for _, entry := range s.Stack {
		if entry.ReturnAddress == Address {
			return true
		}
	}
	return false
}

// Push Subprogram entry
func (s *CallStack) Push(Symbol *dwarf.Entry, ReturnAddress uint64) {

	var NewEntry = &CallStackEntry{
		Self:          Symbol,
		ReturnAddress: ReturnAddress,
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

func (s *CallStack) Last() *CallStackEntry {
	return s.Stack[len(s.Stack)-1]
}

func (Context *DebugContext) GetCurrentReturnAddress() uint64 {
	addr := Context.PeekDataWrapper(uintptr(Context.Target.Regs.Rsp), 8)

	err := Context.SetBreakpoint(uintptr(Context.Target.Regs.Rsp), true)
	if err != nil {
		Context.Logger.Println("Breakpoint already exists at return address, skipping.")
	}

	return binary.LittleEndian.Uint64(addr)
}
