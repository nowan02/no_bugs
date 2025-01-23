package main

import "syscall"

// Sets breakpoint and returns the original data which was replaced.
func SetBreakpoint(pid int, address uintptr) []byte {

	original := []byte{}

	syscall.PtracePeekData(pid, address, original)

	syscall.PtracePokeData(pid, address, []byte{0xCC})

	return original
}

func ReplaceBreakpoint(pid int, address uintptr, registers *syscall.PtraceRegs, data []byte) {
	syscall.PtracePokeData(pid, address, data)

	registers.Rip -= 1

	syscall.PtraceSetRegs(pid, registers)
}
