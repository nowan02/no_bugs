package symbol

//#include <sys/ptrace.h>
//#include <sys/types.h>
//#include <sys/syscall.h>
//#include <stdint.h>
//unsigned long PeekData(int pid, uint64_t rip) {
//	return ptrace(PTRACE_PEEKDATA, pid, rip, NULL);
//}
import "C"

import (
	"syscall"
)

// Wrapper for C function
// Read n bytes of data from location
func PeekDataWrapper(pid int, address uintptr, length int) []byte {
	data := make([]byte, length)

	for i := 0; i < length; i++ {
		data[i] = byte(C.PeekData(C.int(pid), C.uint64_t(address)))
		address++
	}

	return data
}

// Sets breakpoint and saves replaced data to context
func (Context *DebugContext) SetBreakpoint(pid int, address uintptr) {

	original := PeekDataWrapper(pid, address, 1)

	syscall.PtracePokeData(pid, address, []byte{0xCC})

	Context.Breakpoints[address] = original
}

// Replaces INT3, steps over then places the breakpoint back.
// Use this to handle a breakpoint you currently stopped at.
// Returns true if breakpoint exists and was handled, false if the breakpoint does not exist
func (Context *DebugContext) StepOverBreakpoint(pid int, address uintptr, registers *syscall.PtraceRegs) bool {

	data, exists := Context.Breakpoints[address]

	if exists {
		syscall.PtracePokeData(pid, address, data)

		// INT3 stops the program after its evaluation, set back PC with 1 byte after replacement to rerun the correct instruction.
		registers.Rip -= 1

		syscall.PtraceSetRegs(pid, registers)

		syscall.PtraceSingleStep(pid)

		syscall.PtraceGetRegs(pid, registers)

		Context.SetBreakpoint(pid, address)
	}

	return exists
}

// Removes a breakpoint, only use it if the breakpoint was previously handled
// by StepOverBreakpoint or was not hit at all.
// Returns true if breakpoint exists and was handled, false if the breakpoint does not exist
func (Context *DebugContext) RemoveBreakpoint(pid int, address uintptr, registers *syscall.PtraceRegs) bool {
	data, exists := Context.Breakpoints[address]
	if exists {
		syscall.PtracePokeData(pid, address, data)

		delete(Context.Breakpoints, address)
	}
	return exists
}
