package main

//#include <sys/ptrace.h>
//#include <sys/types.h>
//#include <sys/syscall.h>
//#include <stdint.h>
//unsigned long PeekData(int pid, uint64_t rip) {
//	return ptrace(PTRACE_PEEKDATA, pid, rip, NULL);
//}
import "C"

import "syscall"

// Sets breakpoint and saves replaced data to context
func SetBreakpoint(pid int, address uintptr, Context *DebugContext) {

	original := []byte{}

	original = append(original, byte(C.PeekData(C.int(pid), C.uint64_t(address))))

	syscall.PtracePokeData(pid, address, []byte{0xCC})

	Context.breakpoints[address] = original
}

// Replaces INT3, steps over then places the breakpoint back.
// Use this to handle a breakpoint you currently stopped at.
// Returns true if breakpoint exists and was handled, false if the breakpoint does not exist
func StepOverBreakpoint(pid int, address uintptr, registers *syscall.PtraceRegs, Context *DebugContext) bool {

	data, exists := Context.breakpoints[address]

	if exists {
		syscall.PtracePokeData(pid, address, data)

		// INT3 stops the program after its evaluation, set back PC with 1 byte after replacement to rerun the correct instruction.
		registers.Rip -= 1

		syscall.PtraceSetRegs(pid, registers)

		syscall.PtraceSingleStep(pid)

		syscall.PtraceGetRegs(pid, registers)

		SetBreakpoint(pid, address, Context)
	}

	return exists
}

// Removes a breakpoint, only use it if the breakpoint was previously handled
// by StepOverBreakpoint or was not hit at all.
// Returns true if breakpoint exists and was handled, false if the breakpoint does not exist
func RemoveBreakpoint(pid int, address uintptr, registers *syscall.PtraceRegs, Context *DebugContext) bool {
	data, exists := Context.breakpoints[address]
	if exists {
		syscall.PtracePokeData(pid, address, data)

		delete(Context.breakpoints, address)
	}
	return exists
}
