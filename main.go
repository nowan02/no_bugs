package main

import "syscall"

func main() {
	Traced, err := StartProcess("a.out", []string{"a", "b"})

	if err != nil {
		println("Process can't be started. Exiting.")
		return
	}

	state, err := Traced.Wait()

	println(state)

	if err != nil {
		return
	}

	Traced.SetOptions(int(syscall.PTRACE_O_TRACECLONE))

	Traced.SingleStep()

	regs, _ := Traced.GetRegisters()

	// RIP?
	println(uintptr(regs.Rip))
}
