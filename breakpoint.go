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
func SetBreakpoint(pid int, address uintptr, Context *DebugContext) []byte {

	original := []byte{}

	original = append(original, byte(C.PeekData(C.int(pid), C.uint64_t(address))))

	syscall.PtracePokeData(pid, address, []byte{0xCC})

	Context.breakpoints[address] = original

	return original
}

func ReplaceBreakpoint(pid int, address uintptr, registers *syscall.PtraceRegs, Context *DebugContext) {

	data := Context.breakpoints[address]

	syscall.PtracePokeData(pid, address, data)

	registers.Rip -= 1

	syscall.PtraceSetRegs(pid, registers)
}
